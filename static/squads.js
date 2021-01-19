
const app = new Vue({
	el:'#app',
	delimiters: ['[[', ']]'],
	data:{
		squads:[],
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
		})
		.then(res => {
			this.squads = res.data; 
			this.noSquadsAtAll = this.squads.length == 0;	
		})
		.catch(error => {console.log("get-squads failed: " + error)})
	},
	methods: {
		addNewSquad:function() {
			this.addNewSquadMode = true;
		},

		submitNewSquad:function(e) {
			e.preventDefault();

			axios({
				method: 'post',
				url: '/methods/squads',
				data: {
					name: this.squadName,
				}
			})
			.then( res => {
				var squad = res.data;
				squad['name'] = this.squadName;
				squad['membersCount'] = 1;
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
		}
	},
})
