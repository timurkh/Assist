const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			homeCounters:[],
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/me/home`,
		})
		.then(res => {
			this.homeCounters = res.data;
			console.log(this.homeCounters);
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve home dashboard information: " + error.response.data.error;
			this.loading = false;
		})
	},
	methods: {
	},
	mixins: [globalMixin],
}).mount("#app");
