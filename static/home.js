const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squads:[],
			events:[],
			requestsToMe:[],
			myRequests:[],
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
			if(pendingApprove != null && pendingApprove.length > 0){
				let pendingApproveCount = pendingApprove.reduce((a,c) => (a+c.count), 0);
				this.requestsToMe.push({
					name: "Join squads",
					url: "/squads",
					count: pendingApproveCount + " candidate" + (pendingApproveCount > 1?"s":""),
				});
			}
			let appliedParticipants = res.data.appliedParticipants;
			if(appliedParticipants != null && appliedParticipants.length > 0){
				this.requestsToMe.push({
					name: "Participate in events",
					url: "/events",
					count: appliedParticipants.length + " applied",
				});
			}
			this.addQueue(res.data.queuesToApprove, "waiting approve");
			this.addQueue(res.data.queuesToHandle, "to be processed");
			this.myRequests = res.data.userRequests;
			
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve home dashboard information: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		})
	},
	methods: {
		addQueue : function (queues, verb) {
			let queuesCount = 0;
			if(queues != null)
				queuesCount = Object.keys(queues).length;
			if(queuesCount > 0){
				let requestsCount = Object.values(queues).reduce((a,c) => a+c, 0);
				this.requestsToMe.push({
					name: "Requests in " + queuesCount + " queue" + (queuesCount>1?"s":""),
					url: "/requests",
					count: requestsCount + " " + verb,
				});
			}
		},
		getSquadsCount : function() {
			if(this.squads != null)
				return this.squads.reduce((a,c) => a+c, 0);
		},
		getCardBodyClass : function() {
			return "card-body pb-0 ";
		},
		getSquadsClass : function() {
			if(this.getSquadsCount() > 0)
				return this.getCardBodyClass() + "bg-success";
			else
				return this.getCardBodyClass() + "bg-secondary";
		},
		getRequestsToMeClass : function() {

			if(this.requestsToMe != null && this.requestsToMe.length > 0)
				return this.getCardBodyClass() + "bg-danger";
			else
				return this.getCardBodyClass() + "bg-secondary";
		},
		getMyRequestsClass : function() {
			if(this.myRequests != null && this.myRequests.length > 0)
				return this.getCardBodyClass() + "bg-warning";
			else 
				return this.getCardBodyClass() + "bg-secondary";
		},
		getEventsClass : function() {
			if(this.events != null && this.events.length > 0) {
				return this.getCardBodyClass() + "bg-warning";
			} else {
				return this.getCardBodyClass() + "bg-secondary";
			}
		},
	},
	mixins: [globalMixin],
}).mount("#app");
