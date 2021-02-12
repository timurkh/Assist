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
			this.error_message = "Failed to retrieve home dashboard information: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		})
	},
	methods: {
		getSquadsCount : function() {
			return this.homeCounters['squads'].reduce((a,c) => a+c);
		},
		getSquadsClass : function() {
			if(this.getSquadsCount() > 0) {
				return "card-body bg-success";
			} else {
				return "card-body bg-secondary";
			}
		},
		getTodoClass : function() {
			if(this.homeCounters['pendingApprove'] > 0) {
				return "card-body bg-danger";
			} else {
				return "card-body bg-secondary";
			}
		},
		getEventsClass : function() {
			if(false) {
				return "card-body bg-warning";
			} else {
				return "card-body bg-secondary";
			}
		},
	},
	mixins: [globalMixin],
}).mount("#app");
