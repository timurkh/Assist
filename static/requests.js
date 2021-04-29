const app = createApp( {
	delimiters: ['[[', ']]'],
	data:function(){
		return {
			loading:true,
			error_message:"",
			mode:"User",
			getting_more:false,
			queues:{},
			requests:{},
			filter:{},
			moreRequestsAvailable:{},
			newRequest:{},
		};
	},
	computed: {
		queuesToHandleOrApprove : function () {
			return this.queuesToHandle.concat(this.queuesToApprove).sort();
		},
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/users/me/queues`,
		})
		.then(res => {

			this.queues["User"] = res.data.userQueues;
			this.requests["User"] = res.data.userRequests;
			this.queues["Processing"] = res.data.queuesToHandle;
			this.requests["Processing"] = res.data.requestsToHanle;
			this.queues["WaitingApprove"] = res.data.queuesToApprove;
			this.requests["WaitingApprove"] = res.data.requestsToApprove;
			for(m of ["User", "Processing", "WaitingApprove"]) {
				if(this.requests[m] != null) {
					this.moreRequestsAvailable[m] = this.requests[m].length == 10;
					this.requests[m] = this.requests[m].map(x => {x.timeFrom = this.getDurationFrom(new Date(x.time)); return x;});
				}
			}

			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve request queues details: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		});
	},
	methods: {
		userGotNoQueues : function() {
			return this.queues[this.mode] == null || this.queues[this.mode].length == 0;
		},
		getQueues : function() {
			return this.queues[this.mode];
		},
		getRequests : function() {
			return this.requests[this.mode];
		},
		getMoreRequestsAvailable : function() {
			return this.moreRequestsAvailable[this.mode];
		},
		setMode : function(mode) {
			this.mode = mode;
		},
		getMore:function() {
			this.getting_more = true;
			var requests = this.requests[this.mode];
			var lastMember = requests[requests.length-1];
			var url = "requests";
			axios({
				method: 'GET',
				url: `/methods/${url}?from=${lastMember.time}&status=${this.mode}`,
			})
			.then(res => {
				this.requests[this.mode] =  [...requests, ...res.data]; 
				this.moreRequestsAvailable = res.data.length == 10;
				this.getting_more = false;
			})
			.catch(err => {
				this.error_message = "Failed to retrieve requests: " + this.getAxiosErrorMessage(err);
				this.getting_more = false;
			});
		},
		createRequest:function() {
			let request = this.newRequest;

			if(request.queueId != null && request.queueId.length > 0) {
				axios({
					method: 'POST',
					url: `/methods/requests`,
					data: request,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then(res => {
					request.id = res.data.requestId;
					request.status = res.data.status;
					request.time = "Just added";
					this.requests["User"].unshift(request); 
					this.newRequest = {};
				})
				.catch(err => {
					this.error_message = "Failed to create request: " + this.getAxiosErrorMessage(err);
				});
			}
		},
		getRequestStatusText:function(s) {
			switch(s) {
				case 0:
					return "Pending Approve";
				case 1:
					return "Being Processed";
				case 2:
					return "Completed";
				case 3:
					return "Declined";
			}
		},
		getDurationFrom:function(time) {
			let now = new Date();
			let duration = (now.getTime() - time.getTime())/1000;

			let dimensions = ['second', 'minute', 'hour', 'day'];
			let vals = [];
			vals.push(duration % 60);
			duration = (duration / 60) >> 0;
			if(duration > 1) {
				vals.push(duration % 60);
				duration = (duration / 60) >> 0;
				if(duration > 0) {
					vals.push(duration % 24);
					duration = (duration / 24) >> 0;
					if(duration>0)
						vals.push(duration);
				}
			}
			let last = vals.length - 1;
			let text = vals[last] + " " + dimensions[last];
			text += vals[last] > 1 ? "s" : "";

			return text;
		},
	},
	mixins: [globalMixin],
}).mount("#app");
