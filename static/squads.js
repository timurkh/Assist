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
			squadNamePrefix:"",
			userIsAdmin: userIsAdmin,
			currentPage: 0,
			pageSize: 5,
		}
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/me/squads`,
		})
		.then(res => {
			this.own_squads = res.data; 
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve list of squads: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		})
	},
	methods: {
		nextPage:function() {
			  if((this.currentPage*this.pageSize) < this.own_squads.length) this.currentPage++;
		},
		prevPage:function() {
			  if(this.currentPage > 1) this.currentPage--;
		},
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
					pendingApproveCount: 0,
					status: 3
				};
				this.error_message = "";
				this.own_squads[squad.id] = squad;
			})
			.catch(err => {
				if(err.response != null && err.response.status == 409) {
					this.error_message = "This name is already taken, please choose another name.";
				} else {
					this.error_message = "Error while adding new squad: " + this.getAxiosErrorMessage(err);
				}
			});
		},
		deleteSquad:function(id, index) {
			if(confirm(`Please confirm you want to delete squad ${id}`)){
				axios({
					method: 'DELETE',
					url: '/methods/squads/' + id,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					delete(this.own_squads[index]);
				})
				.catch(err => {
					this.error_message = "Error while removing squad " + id + ": " + this.getAxiosErrorMessage(err);
				});
			}
		},
		leaveSquad:function(id, index) {
			if(confirm(`Please confirm you want to leave squad ${id}`)){
				index = index;
				axios({
					method: 'DELETE',
					url: '/methods/squads/' + id + '/members/me',
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					delete this.own_squads[id];
					this.other_squads.push(id);
				})
				.catch(err => {
					this.error_message = "Error while removing squad " + id + ": " + this.getAxiosErrorMessage(err);
				});
			}
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
				var squad = res.data;
				squad.id = id;
				this.own_squads[id] = squad;
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
		showJoinSquadModal:function() {
			axios({
				method: 'GET',
				url: `/methods/squads`,
			})
			.then(res => {
				this.other_squads = res.data; 
				$('#joinSquadModal').modal('show')
			})
			.catch(error => {
				this.error_message = "Failed to retrieve list of squads: " + this.getAxiosErrorMessage(error);
				this.loading = false;
			})
		},
	},
	mixins: [globalMixin],
}).mount("#app");

