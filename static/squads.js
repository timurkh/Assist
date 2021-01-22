
const app = new Vue({
	el:'#app',
	delimiters: ['[[', ']]'],
	data:{
		no_squads:false,
		own_squads:[],
		member_squads:[],
		other_squads:[],
		squadError:"",
		squadName:"",
		squadToJoin:"",
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
			this.member_squads = res.data['Member']; 
			this.other_squads = res.data['Other']; 
			this.no_squads = this.own_squads.length == 0 && this.member_squads.length == 0;
		})
		.catch(error => {console.log("get-squads failed: " + error)})
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
					id: res.data.ID,
					name: this.squadName, 
					membersCount: 1
				};
				this.squadError = "";
				this.own_squads.push(squad);
				this.no_squads = this.own_squads.length == 0 && this.member_squads.length == 0;
			})
			.catch(err => {
				this.squadError = "Error while adding new squad: " + err;
			});
		},
		deleteSquad:function(id, index) {
			axios({
				method: 'delete',
				url: '/methods/squads/' + id,
			})
			.then( res => {
				this.squadError = "";
				this.own_squads.splice(index, 1);
				this.no_squads = this.own_squads.length == 0 && this.member_squads.length == 0;
			})
			.catch(err => {
				this.squadError = "Error while removing squad " + id + ": " + err;
			});
		},
		leaveSquad:function(id, index) {
			index = index;
			axios({
				method: 'delete',
				url: '/methods/squads/' + id + '/members/me',
			})
			.then( res => {
				this.squadError = "";
				var squad = this.member_squads[index];
				squad.membersCount--;
				this.other_squads.push(squad);
				this.member_squads.splice(index, 1);
				this.no_squads = this.own_squads.length == 0 && this.member_squads.length == 0;
			})
			.catch(err => {
				this.squadError = "Error while removing squad " + id + ": " + err;
			});
		},
		joinSquad:function() {
			var id = this.squadToJoin.id;
			var index = this.squadToJoin.index;
			axios({
				method: 'put',
				url: '/methods/squads/' + id + '/members/me',
			})
			.then( res => {
				this.squadError = "";
				var squad = this.other_squads[index];
				squad.membersCount++;
				this.member_squads.push(squad);
				this.other_squads.splice(index, 1);
				this.no_squads = this.own_squads.length == 0 && this.member_squads.length == 0;
			})
			.catch(err => {
				this.squadError = "Error while joining squad: " + err;
			});
		},
	},
})
