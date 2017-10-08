
var secret1 = "yNaS", myTimer, fadeTimeout = 2000;

function ajax() {
	var httpRequest;
	// Old compatibility code, no longer needed.
	if (window.XMLHttpRequest) { // Mozilla, Safari, IE7+ ...
		httpRequest = new XMLHttpRequest();
	} else if (window.ActiveXObject) { // IE 6 and older
		httpRequest = new ActiveXObject("Microsoft.XMLHTTP");
	}
	return httpRequest;
}

function requestResult() {
	if (this.readyState === XMLHttpRequest.DONE) {
		if (this.status === 200) {
			var selRes = document.getElementById("tResult"),
				d = JSON.parse(this.responseText), txt;
			if (d.success) {
				txt = "Success.";
			} else {
				txt = "Failed: " + d.error;
			}
			selRes.innerText = txt;
			console.log("result: ", txt);
			if(myTimer) {
				clearTimeout(myTimer);
			}
			myTimer = setTimeout(function() {
				document.getElementById("tResult").innerText = "";
			}, fadeTimeout);
		} else {
			alert('There was a problem with the request.');
		}
	}
}

function isSamePwd(prefix, res) {
	var newP = document.getElementById(prefix + "tNewPwd").value,
		retypeP = document.getElementById(prefix + "tRetypePwd").value;
	if (newP === retypeP) {
		res.same = true;
		res.password = newP;
		return true;
	}
	return false;
}

function listenButtonClick(prefix, handler) {
	document.getElementById(prefix+"bSubmit").addEventListener('click',
		handler);
}

function adminAuth() {
	var selName = document.getElementById("admintName"),
		selPwd = document.getElementById("admintPwd");
	return {
		adminuser: encodeURIComponent(selName.value),
		adminpassword: sha1(selName.value + ":" + selPwd.value + ":" + secret1)
	};
}

window.onload = function() {
	listenButtonClick('change', modRequest);
	listenButtonClick('add', addRequest);
	listenButtonClick('del', delRequest);

	function modRequest() {
		var req, selName, res = {};
		if (!isSamePwd('change', res)) {
			alert('Password not match.');
			return false;
		}
		req = ajax();
		if (!req) {
			alert('Giving up :( Cannot create an XMLHTTP instance');
			return false;
		}
		selName = document.getElementById("admintName");
		req.open('PATCH', '/rest/admin');
		req.setRequestHeader("Content-type", "application/json");
		req.onreadystatechange = requestResult;
		var d = adminAuth();
		d.newpassword = sha1(selName.value + ":" + res.password + ":" +
			secret1);
		req.send(JSON.stringify(d));
	}

	function addRequest() {
		var req, selName, sha1v, res = {};

		if (!isSamePwd('add', res)) {
			alert('Password not match.');
			return false;
		}
		req = ajax();
		if (!req) {
			alert('Giving up :( Cannot create an XMLHTTP instance');
			return false;
		}
		selName = document.getElementById("addtName");
		sha1v = sha1(selName.value + ":" + res.password + ":" + secret1);
		req.open('POST', '/rest/user');
		req.setRequestHeader("Content-type", "application/json");
		req.onreadystatechange = requestResult;
		var d = adminAuth();
		d.user = selName.value;
		d.password = sha1v;
		if (document.getElementById('addcAdmin').checked) {
			d.isadmin = true;
		}
		req.send(JSON.stringify(d));
	}

	function delRequest() {
		var req, selName;

		req = ajax();
		if (!req) {
			alert('Giving up :( Cannot create an XMLHTTP instance');
			return false;
		}
		selName = document.getElementById("deltName");
		req.open('DELETE', '/rest/user');
		req.setRequestHeader("Content-type", "application/json");
		req.onreadystatechange = requestResult;
		var d = adminAuth();
		d.user = selName.value;
		req.send(JSON.stringify(d));
	}
};
