
const app = new Vue({
	el:'#app',
	delimiters: ['[[', ']]'],
	data:{
		squads:[],
		other_squads:[],
		addNewSquadMode:false,
		squadError:"",
		squadName:"",
		message:"aaa",
		noSquadsAtAll:false
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
			this.squads = res.data['My']; 
			this.other_squads = res.data['Other']; 
			this.noSquadsAtAll = this.squads.length == 0;	
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
				this.addNewSquadMode = false;
				this.squads.push(squad);
				this.noSquadsAtAll = false;
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
				this.addNewSquadMode = false;
				this.squads.splice(index, 1);
				this.noSquadsAtAll = this.squads.length == 0;	
			})
			.catch(err => {
				this.squadError = "Error while removing squad " + id + ": " + err;
			});
		},
		joinSquad:function() {
		},
	},
})
