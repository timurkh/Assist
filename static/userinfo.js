// Disable form submissions if there are invalid fields

function escapeHtml(unsafe) {
	return unsafe
		.replace(/&/g, "&amp;")
		.replace(/</g, "&lt;")
		.replace(/>/g, "&gt;")
		.replace(/"/g, "&quot;")
		.replace(/'/g, "&#039;");
}

firebase.auth().onAuthStateChanged(user => {
	if (user) {

		// init user info settings
		var inputs = document.getElementById('userInfo').getElementsByTagName('input');
		for (var i=0; i<inputs.length; ++i) {
			inputs[i].value = user[inputs[i].id];
		}

		if(document.getElementById('role')) {
			user.getIdTokenResult().then((idTokenResult) => {
					document.getElementById('role').value = idTokenResult.claims.Role; 
			});
		}

		// init auth providers
		var pd = user.providerData;
		for(var i=0; i<pd.length; ++i) {
			document.getElementById(pd[i].providerId).checked = true;
		}
		if(pd.length == 1) {
			document.getElementById(pd[0].providerId).disabled = true;
		}
	}
})

const editInput = function(id) {
	var input = document.getElementById(id);
	if(input.disabled) {
		input.disabled = false;
	} else {
		switch(id){
			case 'displayName':
				firebase.auth().currentUser.updateProfile(
					{displayName: escapeHtml(input.value)}).then(function(){
					}, function(error) {
						document.getElementById(id + 'Error').value = error;
					});
				break;
			case 'email':
				firebase.auth().currentUser.verifyBeforeUpdateEmail(
					escapeHtml(input.value)).then(function(){
						document.getElementById(id + 'Error').textContent = "Please check your inbox. After you complete verification, email setting will updated.";
					}, function(error) {
						document.getElementById(id + 'Error').textContent = error;
					});
				break;
			case 'phoneNumber':
				break;
		}

		document.getElementById(id).disabled = true;
	}

	document.getElementById(id + 'Btn').getElementsByTagName('i')[0].classList.toggle("fa-pen");
	document.getElementById(id + 'Btn').getElementsByTagName('i')[0].classList.toggle("fa-check");
}

const toggleIDProvider = function(checkBox) {
	var user = firebase.auth().currentUser;
	if(checkBox.checked) {
		var func, arg;
		switch(checkBox.id) {
			case "password":
				arg = firebase.auth.EmailAuthProvider.credential(user.email, Math.random().toString(36).slice(-8));
				func = 'linkWithCredential';
				break;
			case "facebook":
				arg = new firebase.auth.FacebookAuthProvider();
				func = 'linkWithPopup';
				break;
			case "google":
				arg = new firebase.auth.GoogleAuthProvider();
				func = 'linkWithPopup';
				break;
		}

		user[func](arg).then (function(){
					}, function(error) {
						document.getElementById(checkBox.id + 'Error').textContent = error;
						checkBox.checked = false;
					});
	} else {
		user.unlink(checkBox.id).then (function(){
					}, function(error) {
						document.getElementById(checkBox.id + 'Error').textContent = error;
					});
	}

	checkIfLastProviderLeft();
}

const checkIfLastProviderLeft = function() {

	var lastCheckedOne = -1;
	var moreThanOne = false;
	var inputs = document.getElementById('providers').getElementsByTagName('input');
	for (var i=0; i<inputs.length; ++i) {
		if(inputs[i].type == "checkbox") {
			if(inputs[i].checked) {
				if( lastCheckedOne != -1) {
					//we have more than one, enable all
					moreThanOne = true;
				} else {
					lastCheckedOne = i;
				}
			}
			inputs[i].disabled = false;
		}
	}

	if(moreThanOne == false) {
		inputs[lastCheckedOne].disabled = true;
	}
}
	
