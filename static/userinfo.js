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
				//console.log(pd[i].providerId);
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
				var name = input.value
				if(name.length) {
					axios({
						method: 'PUT',
						url: `/methods/users/me`,
						data: {
							name: name,
						}
					})
					.then( function() {
						user.updateProfile( {displayName: escapeHtml(input.value)})
					}, function(error) {
						document.getElementById(id + 'Error').textContent = error.response.data;
					})
					.then( function() {
						//name updated successfully
					})
					.catch( error => {
						document.getElementById(id + 'Error').textContent = error;
					});
				} else {
					document.getElementById(id + 'Error').textContent = "Please specify non-empty name.";
				}
				break;
			case 'email':
				var email = input.value;
				if (email.length) {
					user.verifyBeforeUpdateEmail(
						escapeHtml(input.value)).then(function(){
							document.getElementById(id + 'Error').textContent = "Please check your inbox. After you complete verification, email setting will be updated.";
						}, function(error) {
							document.getElementById(id + 'Error').textContent = error;
						});
				} else {
					document.getElementById(id + 'Error').textContent = "Please specify valid email.";
					input.value = user.email;
				}
				break;
			case 'phoneNumber': 
				document.getElementById(id + 'Error').textContent = "";
				var provider = new firebase.auth.PhoneAuthProvider();
				var phoneNumber = input.value;

				if (phoneNumber.length == 0) {
					user.unlink("phone").then(function() {
						document.getElementById('phone').checked = false;
						document.getElementById(id + 'Error').textContent = "";
					}).catch(function(error) {
						document.getElementById(id + 'Error').textContent = "Failed to unlink phone from account: " + error;
					});
				} else {
					provider.verifyPhoneNumber(input.value, appVerifier)
						.then( function(verificationId) {
							var verificationCode = window.prompt('Please enter the verification ' +
																 'code that was sent to your mobile device.');
							var phoneCredential = firebase.auth.PhoneAuthProvider.credential(verificationId, verificationCode);
							return user.updatePhoneNumber(phoneCredential);
						})
						.then((result) => {
							// Phone credential now linked to current user.
							document.getElementById('phone').checked = true;
							appVerifier.reset();
						})
						.catch((error) => {
							// Error occurred.
							document.getElementById(id + 'Error').textContent = "Failed to validate that you own provided phone number: " + error;
							appVerifier.reset();
						})
				}
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

			if(inputs[i].id != "phone") { 
				// phone checkbox is always disabled
				// to unlink phone just set empty number
				inputs[i].disabled = false;
			}
		}
	}

	if(moreThanOne == false) {
		inputs[lastCheckedOne].disabled = true;
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

// init appVerifier
var appVerifier;
window.addEventListener('load', function() {
	appVerifier = new firebase.auth.RecaptchaVerifier( "recaptcha", { size: "invisible" });
});

	
const sendVerificationEmail = function(e) {
	e.preventDefault();
	user = firebase.auth().currentUser;
	user.sendEmailVerification().then(
		function() {
			document.getElementById('emailError').textContent = 'Verification email sent. Please check your inbox and follow instructions.';
		}
	).catch(error => {
		document.getElementById('emailError').textContent = "Failed to send verification email: " + error + ". You can either change email or refresh this page and try once more."
	});
}
