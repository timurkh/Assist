const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squadsCount:0,
			pendingApprove:0
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/{userId}/home`,
		})
		.then(res => {
			this.squad_members = res.data['squadsCount']; 
			this.squad_owner = res.data['pendingApprove'];
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve home dashboard information: " + error;
			this.loading = false;
		})
	},
	methods: {
	},
	mixins: [globalMixin],
}).mount("#app");
