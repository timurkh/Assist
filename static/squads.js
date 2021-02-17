const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			own_squads:[],
			other_squads:[],
			error_message:"",
			squadName:"",
			squadToJoin:"",
			userIsAdmin: userIsAdmin,
		}
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/me/squads`,
		})
		.then(res => {
			this.own_squads = res.data['Own']; 
			this.other_squads = res.data['Other']; 
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve list of squads: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		})
	},
	methods: {
		submitNewSquad:function() {

			axios({
				method: 'POST',
				url: '/methods/squads',
				data: {
					name: this.squadName,
				},
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				var squad = {
					id: this.squadName, 
					membersCount: 1,
					status: 3
				};
				this.error_message = "";
				this.own_squads.push(squad);
			})
			.catch(err => {
				if(err.response.status == 409) {
					this.error_message = "This name is already taken, please choose another name.";
				} else {
					this.error_message = "Error while adding new squad: " + this.getAxiosErrorMessage(err);
				}
			});
		},
		deleteSquad:function(id, index) {
			axios({
				method: 'DELETE',
				url: '/methods/squads/' + id,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.own_squads.splice(index, 1);
			})
			.catch(err => {
				this.error_message = "Error while removing squad " + id + ": " + this.getAxiosErrorMessage(err);
			});
		},
		leaveSquad:function(id, index) {
			index = index;
			axios({
				method: 'DELETE',
				url: '/methods/squads/' + id + '/members/me',
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				var squad = this.own_squads[index];
				squad.membersCount--;
				this.other_squads.push(squad);
				this.own_squads.splice(index, 1);
			})
			.catch(err => {
				this.error_message = "Error while removing squad " + id + ": " + this.getAxiosErrorMessage(err);
			});
		},
		joinSquad:function() {
			var id = this.squadToJoin.id;
			var index = this.squadToJoin.index;
			axios({
				method: 'POST',
				url: '/methods/squads/' + id + '/members/me',
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				var squad = this.other_squads[index];
				squad.status = res.data.Status;
				squad.membersCount++;
				this.own_squads.push(squad);
				this.other_squads.splice(index, 1);
			})
			.catch(err => {
				this.error_message = "Error while joining squad: " + this.getAxiosErrorMessage(err);;
			});
		},
		showSquadDetails:function(squadId, index) {
			window.location.href = `/squads/` + encodeURI(squadId);
		},
		showSquadMembers:function(squadId, index) {
			window.location.href = `/squads/` + encodeURI(squadId) + `/members`;
		},
	},
	mixins: [globalMixin],
}).mount("#app");

