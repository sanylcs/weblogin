
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

window.onload = function() {
	document.getElementById("btnSubmit").addEventListener('click',
		makeRequest);

	function makeRequest() {
		var req, selName = document.getElementById("txtName"),
			selPwd = document.getElementById("txtPwd"),
			sha1v = sha1(selName.value + ":" + selPwd.value + ":" + secret1);

		req = ajax();
		if (!req) {
			alert('Giving up :( Cannot create an XMLHTTP instance');
			return false;
		}
		req.open('POST', '/rest/access');
		req.setRequestHeader("Content-type", "application/json");
		req.onreadystatechange = requestResult;
		var d = {user: encodeURIComponent(selName.value), password: sha1v};
		req.send(JSON.stringify(d));
	}
};
