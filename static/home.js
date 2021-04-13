const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squads:[],
			events:[],
			requestsToMe:[],
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/me/home`,
		})
		.then(res => {
			this.squads = res.data.squads;
			this.events = res.data.events.map(x => {x.date = new Date(x.date); return x});
			this.eventsCount = res.data.eventsCount;

			let pendingApprove = res.data.pendingApprove;
			if(pendingApprove != null & pendingApprove.length > 0){
				this.requestsToMe.push({
					name: "Join squads",
					url: "/squads",
					count: pendingApprove.length + " candidate" + (pendingApprove.length > 1?"s":""),
				});
			}
			let appliedParticipants = res.data.appliedParticipants;
			if(appliedParticipants != null & appliedParticipants.length > 0){
				this.requestsToMe.push({
					name: "Participate in events",
					url: "/events",
					count: appliedParticipants.length + " applied",
				});
			}
			//let queuesToApprove = res.data.queuesToApprove.filter(queue => queue.requestsWaitingApprove.length > 0);
			let queuesToApprove = res.data.queuesToApprove;
			if(queuesToApprove != null & queuesToApprove.length > 0){

				this.requestsToMe.push({
					name: queuesToApprove.length > 1 ? queuesToApprove.length + " queues" :  queuesToApprove[0].id,
					url: "/requests",
					count: queuesToApprove.reduce((a,c) => a+c.requestsWaitingApprove, 0) + " requests waiting approve",
				});
			}

			//let queuesToHandle = res.data.queuesToHandle.filter(queue => queue.requestsProcessing.length > 0);
			let queuesToHandle = res.data.queuesToHandle;
			if(queuesToHandle != null & queuesToHandle.length > 0){

				this.requestsToMe.push({
					name: queuesToHandle.length > 1 ? queuesToHandle.length + " queues" : queueusToHandle[0].id,
					url: "/requests",
					count: queuesToHandle.reduce((a,c) => a+c.requestsProcessing, 0) + " requests to be processed",
				});
			}
			
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
				return this.squads.reduce((a,c) => a+c, 0);
		},
		getSquadsClass : function() {
			if(this.getSquadsCount() > 0) {
				return "card-body bg-success";
			} else {
				return "card-body bg-secondary";
			}
		},
		getRequestsToMeClass : function() {
			if(this.requestsToMe.length > 0)
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
