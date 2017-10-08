package main

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	dbPath  = "db"
	wwwPath = "www"
	secret2 = "WeIlYnAs"
	fpermit = 0644
)

var (
	errNoMatch = errors.New("username and password not match")
	errNoUser  = errors.New("user not found")
)

func main() {
	http.HandleFunc("/hello", helloServer)
	http.HandleFunc("/rest/access", accessHandler)
	http.HandleFunc("/rest/admin", adminHandler)
	http.HandleFunc("/rest/user", userHandler)
	http.Handle("/", http.FileServer(http.Dir(wwwPath)))
	log.Fatal(http.ListenAndServe(":80", nil))
}

// hello world, the web server
func helloServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

type adminType struct {
	AdminUser     string `json:"adminuser"`
	AdminPassword string `json:"adminpassword"`
	NewPassword   string `json:"newpassword"`
}

type userType struct {
	adminType
	User     string `json:"user"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"isadmin"`
}

type requestResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func unmarshal(req *http.Request, v interface{}) error {
	err := json.NewDecoder(req.Body).Decode(v)
	err2 := req.Body.Close()
	if err != nil && err2 != nil {
		return fmt.Errorf("%v %v", err, err2)
	} else if err2 != nil {
		return err2
	}
	return err
}

func accessHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	var v userType
	if err := unmarshal(req, &v); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user := v.User
	if user == "" {
		http.Error(w, "Invalid username value.", http.StatusBadRequest)
		return
	}
	err := isValidUser(user, v.Password, false)
	if err == nil {
		err = enableAction()
	}
	errHandler(w, err)
}

func adminHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if req.Method != http.MethodPatch {
		http.Error(w, "Unsupported method", http.StatusBadRequest)
		return
	}
	var v adminType
	if err := isAdminAuthorized(req, &v); err != nil {
		http.Error(w, fmt.Sprint("Failed authorized: ", err.Error()),
			http.StatusBadRequest)
		return
	}
	np := v.NewPassword
	if np == "" {
		http.Error(w, "Invalid password value.", http.StatusBadRequest)
		return
	}
	// if authorized then admin user surely non-empty
	errHandler(w, delOrReplacePwd(v.AdminUser, np, true, true))
}

func errHandler(w http.ResponseWriter, err error) {
	var v requestResult
	if err != nil {
		v.Error = err.Error()
	} else {
		v.Success = true
	}
	bs, err2 := json.Marshal(&v)
	if err2 != nil {
		if err == nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			http.Error(w, fmt.Sprint(err2.Error(), " ", err.Error()),
				http.StatusInternalServerError)
		}
		return
	}
	w.Write(bs)
}

func userHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if req.Method != http.MethodPost && req.Method != http.MethodDelete {
		http.Error(w, "Unsupported method", http.StatusBadRequest)
		return
	}
	var v userType
	err := isUserAuthorized(req, &v)
	if err != nil {
		http.Error(w, fmt.Sprint("Failed authorized: ", err.Error()),
			http.StatusBadRequest)
		return
	}
	user := v.User // if authorized this surely non-empty
	if req.Method == http.MethodPost {
		err = addUser(user, v.Password, v.IsAdmin)
	} else {
		err = delOrReplacePwd(user, "", false, false)
	}
	errHandler(w, err)
}

func isAdminAuthorized(req *http.Request, admin *adminType) error {
	if err := unmarshal(req, admin); err != nil {
		return err
	}
	return isAuthorized(req, admin)
}

func isUserAuthorized(req *http.Request, user *userType) error {
	if err := unmarshal(req, user); err != nil {
		return err
	}
	return isAuthorized(req, &user.adminType)
}

func isAuthorized(req *http.Request, admin *adminType) error {
	user := admin.AdminUser
	if user == "" {
		return errors.New("Invalid admin username value.")
	}
	return isValidUser(user, admin.AdminPassword, true)
}

func closeFile(c io.Closer, err error) error {
	if cer := c.Close(); cer != nil {
		return errors.New(err.Error() + " " + cer.Error())
	}
	return err
}

func addUser(usr, np string, isAdmin bool) error {
	file, err := getFile(isAdmin, false, true)
	if err != nil {
		return err
	}
	if err = isUserExist(usr, "", file); err == nil || err == errNoMatch {
		return closeFile(file, errors.New("user already exist"))
	} else if err != errNoUser {
		return closeFile(file, err)
	}
	rec := [2]string{usr, np}
	w := csv.NewWriter(file)
	if err = w.Write(rec[:]); err != nil {
		return closeFile(file, err)
	}
	w.Flush()
	if err = w.Error(); err != nil {
		return closeFile(file, err)
	}
	return file.Close()
}

func delOrReplacePwd(usr, np string, isAdmin, replacePwd bool) error {
	file, err := getFile(isAdmin, false, false)
	if err != nil {
		return err
	}
	recs, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return closeFile(file, err)
	}
	var (
		found bool
		index int
	)
	for idx, rec := range recs {
		if strings.EqualFold(rec[0], usr) {
			found = true
			index = idx
			break
		}
	}
	if !found {
		var s string
		if replacePwd {
			s = " for password replacement"
		}
		return closeFile(file, errors.New("can't find user: "+usr+s))
	}
	if replacePwd {
		bs := sha1.Sum([]byte(fmt.Sprint(np, ":", secret2)))
		recs[index][1] = hex.EncodeToString(bs[:])
	} else {
		for i := index; i < len(recs)-1; i++ {
			recs[i] = recs[i+1]
		}
		recs = recs[:len(recs)-1]
	}
	if _, err = file.Seek(0, 0); err != nil {
		return closeFile(file, err)
	}
	if err = file.Truncate(0); err != nil {
		return closeFile(file, err)
	}
	if err = csv.NewWriter(file).WriteAll(recs); err != nil {
		return closeFile(file, err)
	}
	return file.Close()
}

func getFile(isAdmin, readOnly, append bool) (*os.File, error) {
	var flg int
	if readOnly {
		flg = os.O_RDONLY
	} else if !append {
		flg = os.O_RDWR
	} else {
		flg = os.O_RDWR | os.O_APPEND
	}
	if isAdmin {
		return os.OpenFile(path.Join(dbPath, "admin.csv"), flg, fpermit)
	}
	return os.OpenFile(path.Join(dbPath, "users.csv"), flg, fpermit)
}

func isValidUser(usr, pwd string, isAdmin bool) error {
	file, err := getFile(isAdmin, true, false)
	if err != nil {
		return err
	}
	if err = isUserExist(usr, pwd, file); err != nil {
		if err == io.EOF {
			return closeFile(file, errNoUser)
		}
		return closeFile(file, err)
	}
	return file.Close()
}

func isUserExist(usr, pwd string, file *os.File) error {
	r := csv.NewReader(file)
	r.ReuseRecord = true
	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return errNoUser
			}
			return err
		}
		if strings.EqualFold(rec[0], usr) {
			bs := sha1.Sum([]byte(fmt.Sprint(pwd, ":", secret2)))
			if !strings.EqualFold(rec[1], hex.EncodeToString(bs[:])) {
				return errNoMatch
			}
			break
		}
	}
	return nil
}

func enableAction() error {
	return nil
}
