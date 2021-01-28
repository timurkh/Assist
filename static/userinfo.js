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
			try {
				document.getElementById(pd[i].providerId).checked = true;
			} catch (error) {
				console.log("Error while processing provider " + pd[i].providerId + ": " + error);
			}
		}
		if(pd.length == 1) {
			try {
				document.getElementById(pd[0].providerId).disabled = true;
			} catch (error) {
				console.log("Error while disabling provider " + pd[i].providerId + ": " + error);
			}
		}
		disablePhoneCheckbox();
	}
})

const editInput = function(id) {
	var input = document.getElementById(id);
	var user = firebase.auth().currentUser;
	if(input.disabled) {
		input.disabled = false;
	} else {
		document.getElementById(id + 'Error').textContent = '';
		switch(id){
			case 'displayName':
				user.updateProfile(
					{displayName: escapeHtml(input.value)}).then(function(){
					}, function(error) {
						document.getElementById(id + 'Error').textContent = error;
					});
				break;
			case 'email':
				user.verifyBeforeUpdateEmail(
					escapeHtml(input.value)).then(function(){
						document.getElementById(id + 'Error').textContent = "Please check your inbox. After you complete verification, email setting will updated.";
					}, function(error) {
						document.getElementById(id + 'Error').textContent = error;
					});
				break;
			case 'phoneNumber':
				var appVerifier = new firebase.auth.RecaptchaVerifier( "recaptcha", { 
					'size': "invisible",
					'callback': function(response) {
						console.log('callback executed');
						console.log(response);
					},
					'expired-callback': function() {
						console.log('expired');
					},
				});
				console.log(appVerifier);
				var provider = new firebase.auth.PhoneAuthProvider();
				provider.verifyPhoneNumber(input.value, appVerifier)
					.then(function (verificationId) {
						var verificationCode = window.prompt('Please enter the verification ' +
															 'code that was sent to your mobile device.');
						phoneCredential = firebase.auth.PhoneAuthProvider.credential(verificationId, verificationCode);
						user.updatePhoneNumber(phoneCredential);
					})
					.then((result) => {
						// Phone credential now linked to current user.
						// User now can sign in with new phone upon logging out.
						console.log(result);
					})
					.catch((error) => {
						// Error occurred.
						document.getElementById(id + 'Error').textContent = "Failed to validate that you are a human: " + error;
					})
				appVerifier.clear();
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
			case "facebook.com":
				arg = new firebase.auth.FacebookAuthProvider();
				func = 'linkWithPopup';
				break;
			case "google.com":
				arg = new firebase.auth.GoogleAuthProvider();
				func = 'linkWithPopup';
				break;
			case "phone":
				console.log("Phone checkbox should not be enabled when phone is not available");	
				checkBox.checked = false;
				checkBox.disabled = true;
				return;
		}

		document.getElementById(checkBox.id + 'Error').textContent = "";
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
	disablePhoneCheckbox();
}

// phone is not OAuth provider, we can only disable it
const disablePhoneCheckbox = function() {
	var phoneCheckbox = document.getElementById('phone');
	if(phoneCheckbox.checked == false) {
		phoneCheckbox.disabled = true;
	}
}

const sendPasswordReset = function() {
	document.getElementById('passwordNotification').value = "";
	document.getElementById('passwordError').value = "";

	var auth = firebase.auth();
	auth.sendPasswordResetEmail(auth.currentUser.email).then(function() {
		document.getElementById('passwordNotification').textContent = "Email with instructions sent to " + auth.currentUser.email + ", please check your inbox";
	}).catch(function(error) {
		document.getElementById('passwordError').textContent = error;
	});
}
	
