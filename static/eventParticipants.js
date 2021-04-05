const app = createApp( {
	delimiters: ['[[', ']]'],
	components: {
		'add-participant-dialog' : AddParticipantDialog,
		'change-status-dialog' : ChangeStatusDialog,
	},
	data:function(){
		return {
			loading:true,
			error_message:"",
			eventId:eventId,
			eventParticipants:[],
			participantToChange: [],
			tags:[],
			evnt:{},
			getting_more:false,
			filter:{ },
			moreRecordsAvailable: false,
			candidates:[],
			prevKeys:"",
			loadingMore:false,
			noMoreCandidates:false,
		};
	},
	created:function() {
		let uri = window.location.search.substring(1); 
		let params = new URLSearchParams(uri);
		this.filter.status = params.get("status");

		axios.all([
			axios.get(`/methods/events/${eventId}`),
			axios.get(`/methods/events/${eventId}/participants`, {params : this.filter}),
			axios.get(`/methods/events/${eventId}/candidates`, {params : this.filter}),
		])
		.then(axios.spread((evnt, participants, candidates) => {
			this.evnt = evnt.data.event;
			this.evnt.date = new Date(this.evnt.date);
			this.tags = evnt.data.tags;
			this.moreRecordsAvailable = participants.data.length == 10;
			this.eventParticipants = participants.data; 
			this.candidates = candidates.data;
			this.noMoreCandidates = candidates.data.length < 10;
			this.loading = false;
		}))
		.catch(err => {
			this.error_message = "Failed to retrieve event participants and tags: " + this.getAxiosErrorMessage(err);
			this.loading = false;
		});
	},
	methods: {

		changeStatus:function(member, index) {
			this.participantToChange = member;
			this.participantToChange.index = index;
			$('#participantToChangeStatusModal').modal('show')
		},
		setParticipantStatus:function(status) {
			axios({
				method: 'PATCH',
				url: `/methods/events/${eventId}/participants/${this.participantToChange.id}`,
				data: { status: status, },
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.eventParticipants[this.participantToChange.index].status = status;
			})
			.catch(err => {
				this.error_message = "Error while changing member status: " + this.getAxiosErrorMessage(err);
			});
		},
		removeParticipant:function(user, index) {
			index = index;
			if(confirm(`Please confirm you want to remove user ${user.displayName} from event '${this.evnt.text}'`)) {
				axios({
					method: 'DELETE',
					url: `/methods/events/${eventId}/participants/${user.id}`,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					this.eventParticipants.splice(index, 1);
				})
				.catch(err => {
					this.error_message = `Error while removing user ${user.displayName} from event: ` + this.getAxiosErrorMessage(err);
				});
			}
		},
		getMore:function() {
			this.getting_more = true;
			let lastMember = this.eventParticipants[this.eventParticipants.length-1];
			axios({
				method: 'GET',
				url: `/methods/events/${eventId}/participants?from=${lastMember.timestamp}`,
				params: this.filter,
			})
			.then(res => {
				this.moreRecordsAvailable = res.data.length == 10;
				this.eventParticipants =  [...this.eventParticipants, ...res.data]; 
				this.getting_more = false;
			})
			.catch(err => {
				this.error_message = "Failed to retrieve squad members and tags: " + this.getAxiosErrorMessage(err);
				this.getting_more = false;
			});
		},
		onFilterChange:function(e) {
			this.loading = true;


			// unfortunately due to firestore limitations I canot search by keys and tag at the same moment :(
			// only one array-in is allowed
			if(e.target.id == "searchKeys")
				this.filter.tag = "";
			else if(e.target.id == "selectTag")
				this.filter.keys = "";
			
			axios({
				method: 'GET',
				url: `/methods/events/${eventId}/participants`,
				params: this.filter, 
			})
			.then( res => {
				this.error_message = "";
				this.moreRecordsAvailable = res.data.length == 10;
				this.eventParticipants = res.data; 
				this.loading = false;
			})
			.catch(err => {
				this.error_message = "Failed to retrieve event participants: " + this.getAxiosErrorMessage(err);
				this.loading = false;
			});
		},
		onCandidateFilterChange : function(e) {
			if(this.prevKeys != e) {
				axios.get(`/methods/events/${eventId}/candidates`, {params : {tags: this.filter.tags, keys: e}})
				.then(res => {
					this.candidates = res.data;
					this.noMoreCandidates = res.data.length < 10;;
					this.prevKeys = e;
				})
				.catch(err => {
					this.error_message = "Failed to retrieve event candidates: " + this.getAxiosErrorMessage(err);
				});
			}
		},
		onCandidateLoadMore : function(e) {
			if(!this.loadingMore && !this.noMoreCandidates) {
				let scrollTop = e.target.scrollTop;
				this.loadingMore = true;
				var lastCandidate = this.candidates[this.candidates.length - 1];
				axios.get(`/methods/events/${eventId}/candidates`, {params : {from: lastCandidate.id, tags: this.filter.tags, keys: this.prevKeys}})
				.then(res => {
					this.noMoreCandidates = res.data.length < 10;
					this.candidates = this.candidates.concat(res.data);
					this.$nextTick(() => {
						e.target.scrollTop = scrollTop;
					});
					this.loadingMore = false;
				})
				.catch(err => {
					this.error_message = "Failed to retrieve event candidates: " + this.getAxiosErrorMessage(err);
				});
			}
		},
		addParticipant : function (e) {
			let users = e.map(u => u.id).join(",");
			axios({
				method: 'POST',
				url: `/methods/events/${eventId}/participants/${users}`,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				var participants = e.map( u => { u.status = res.data.status; return u;}); 
				this.eventParticipants = this.eventParticipants.concat(participants);
			})
			.catch(err => {
				this.error_message = "Error while adding event participants: " + this.getAxiosErrorMessage(err);
			});
		},
	},
	mixins: [globalMixin],
}).mount("#app");
