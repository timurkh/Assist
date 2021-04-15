const app = createApp( {
	delimiters: ['[[', ']]'],
	data:function(){
		return {
			loading:true,
			error_message:"",
			mode:"user",
			requests:[],
			getting_more:false,
			filter:{ },
			filterToHandle:{ },
			myQueues:[],
			queuesToHandle:[],
			queuesToApprove:[],
			myRequests:[],
			requestsToHandle:[],
			requestsToApprove:[],
			moreRequestsAvailable: false,
		};
	},
	computed: {
		queuesToHandleOrApprove : function () {
			return this.queuesToHandle.concat(this.queuesToApprove).sort();
		},
	},
	created:function() {
		axios.all([
			axios.get(`/methods/users/me/queues`),
		])
		.then(axios.spread((queues) => {
			console.log(queues.data);
			this.queuesToHandle = queues.data.queuesToHandle;
			this.queuesToApprove = queues.data.queuesToApprove;
			this.myQueues = queues.data.userQueues;
			this.myRequests = queues.data.userRequests;
			this.requestsToHandle = queues.data.requestsToHandle;
			this.requestsToApprove = queues.data.requestsToApprove;

			this.loading = false;
		}))
		.catch(errors => {
			this.error_message = "Failed to retrieve request queues details: " + this.getAxiosErrorMessage(errors);
			this.loading = false;
		});
	},
	methods: {
		userGotNoQueues : function() {
			switch(this.mode) {
				case "handle":
					return this.queuesToHandle == null || this.queuesToHandle.length == 0;
				case "approve":
					return this.queuesToApprove == null || this.queuesToApprove.length == 0;
				default:
					return this.myQueues == null || this.myQueues.length == 0
			}
		},
		getQueues : function() {
			switch(this.mode) {
				case "handle":
					return this.queuesToHandle;
				case "approve":
					return this.queuesToApprove;
				default:
					return this.myQueues;
			}
		},
		setMode : function(mode) {
			this.mode = mode;
		},
		getMore:function() {
			this.getting_more = true;
			let lastMember = this.squad_members[this.squad_members.length-1];
			axios({
				method: 'GET',
				url: `/methods/squads/${squadId}/members?from=${lastMember.timestamp}`,
				params: this.filter,
			})
			.then(res => {
				this.moreRecordsAvailable = res.data.length == 10;
				this.requests =  [...this.requests, ...res.data]; 
				this.getting_more = false;
			})
			.catch(err => {
				this.error_message = "Failed to retrieve squad members and tags: " + this.getAxiosErrorMessage(err);
				this.getting_more = false;
			});
		},
		onFilterChange:function(e) {
		},
	},
	mixins: [globalMixin],
}).mount("#app");
