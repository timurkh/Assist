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
		}
	},
	created:function() {
		axios({
			method: 'GET',
			url: '/methods/squads',
			params: {
				userId : 'me' 
			}
		})
		.then(res => {
			this.own_squads = res.data['Own']; 
			this.other_squads = res.data['Other']; 
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve list of squads: " + error;
			this.loading = false;
		})
	},
	methods: {
		submitNewSquad:function() {

			axios({
				method: 'post',
				url: '/methods/squads',
				data: {
					name: this.squadName,
				}
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
					this.error_message = "Error while adding new squad: " + err;
				}
			});
		},
		deleteSquad:function(id, index) {
			axios({
				method: 'delete',
				url: '/methods/squads/' + id,
			})
			.then( res => {
				this.error_message = "";
				this.own_squads.splice(index, 1);
			})
			.catch(err => {
				this.error_message = "Error while removing squad " + id + ": " + err;
			});
		},
		leaveSquad:function(id, index) {
			index = index;
			axios({
				method: 'delete',
				url: '/methods/squads/' + id + '/members/me',
			})
			.then( res => {
				this.error_message = "";
				var squad = this.own_squads[index];
				squad.membersCount--;
				this.other_squads.push(squad);
				this.own_squads.splice(index, 1);
			})
			.catch(err => {
				this.error_message = "Error while removing squad " + id + ": " + err;
			});
		},
		joinSquad:function() {
			var id = this.squadToJoin.id;
			var index = this.squadToJoin.index;
			axios({
				method: 'POST',
				url: '/methods/squads/' + id + '/members/me',
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
				this.error_message = "Error while joining squad: " + err;
			});
		},
		showSquadDetails:function(squadId, index) {
			window.location.href = `/squad?squadId=` + encodeURI(squadId);
		},
	},
	mixins: [globalMixin],
}).mount("#app");

