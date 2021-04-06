const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squads:[],
			pendingApprove:[],
			events:[],
			queuesToApprove:[],
			queuesToHandle:[],
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
			this.queuesToApprove = res.data.queuesToApprove;
			//this.queuesToApprove = res.data.queuesToApprove.filter(queue => queue.requestsWaitingApprove.length > 0);
			this.queuesToHandle = res.data.queuesToHandle;
			//this.queuesToHandle = res.data.queuesToHandle.filter(queue => queue.requestsProcessing.length > 0);
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve home dashboard information: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		})
	},
	methods: {
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
		getTodoCount : function () {
			var count = 0;
			console.log(this.pendingApprove);
			if (this.pendingApprove != null && this.pendingApprove.length>0)
				count = this.pendingApprove.reduce((count,s) => count+s.count, count);
			if (this.queuesToApprove!=null && this.queuesToApprove.length>0)
				count = this.queuesToApprove.reduce((count,q) => count+q.requestsWaitingApprove, count);
			if (this.queuesToHandle!=null && this.queuesToHandle.length>0)
				count = this.queuesToHandle.reduce((count,q) => count+q.requestsProcessing, count);
			return count;
		},
		getTodoClass : function() {
			if(this.getTodoCount() > 0)
				return "card-body bg-danger";
			return "card-body bg-secondary";
		},
		getMyRequestsClass : function() {
			return "card-body bg-secondary";
		},
		getEventsClass : function() {
			if(this.events != null && this.events.length > 0) {
				return "card-body bg-warning";
			} else {
				return "card-body bg-secondary";
			}
		},
	},
	mixins: [globalMixin],
}).mount("#app");
