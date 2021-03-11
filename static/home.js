const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squads:[],
			pendingApprove:[],
			events:[],
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/me/home`,
		})
		.then(res => {
			this.squads = res.data.squads;
			this.pendingApprove = res.data.pendingApprove;
			this.events = res.data.events.map(x => {x.date = new Date(x.date); return x});
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve home dashboard information: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		})
	},
	methods: {
		getDate(date) {
			return date.toLocaleString('ru', {
				    day:   '2-digit',
				    month: '2-digit',
				    year:  '2-digit'
				  });
		},
		getSquadsCount : function() {
			if(this.squads != null)
				return this.squads.reduce((a,c) => a+c);
		},
		getSquadsClass : function() {
			if(this.getSquadsCount() > 0) {
				return "card-body bg-success";
			} else {
				return "card-body bg-secondary";
			}
		},
		getTodoClass : function() {
			let hc = this.pendingApprove;
			if(hc != null && hc.length > 0) {
				return "card-body bg-danger";
			} 

			return "card-body bg-secondary";
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
