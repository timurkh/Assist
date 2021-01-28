const app = new Vue({
	el:'#app',
	delimiters: ['[[', ']]'],
	data:{
		loading:true,
		error_message:"",
		squad_members:[],
		squad_owner:[],
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/squads/${squadId}/members`,
		})
		.then(res => {
			this.squad_members = res.data['Members']; 
			this.squad_owner = res.data['Owner'];
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve list of squad members: " + error;
			this.loading = false;
		})
	},
	methods: {
		submitNewSquad:function() {
		},
	},
})
