const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	components: {
		'add-event-dialog' : AddEventDialog,
	},
	data(){
		return {
			loading:true,
			userIsAdmin: userIsAdmin,
			newEvnt:{},
			squads:{},
			events:[],
			currentUserId:currentUserId,
		}
	},
	created:function() {
		axios.all([
			axios.get(`/methods/users/me/squads?status=admin`),
			axios.get(`/methods/users/me/events`),
		])
		.then(axios.spread((squads, events) => {
			this.squads = squads.data;
			this.events = events.data.map(x => {x.date = new Date(x.date); return x});
			this.loading = false;
		}))
		.catch(errors => {
			this.error_message = "Failed to retrieve events: " + this.getAxiosErrorMessage(errors);
			this.loading = false;
		});
	},
	methods: {
		addEvent:function(e) {
			e.date = new Date(e.date);
			axios({
				method: 'POST',
				url: '/methods/events',
				data: e,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				e.id = res.data.id;
				e.ownerId = currentUserId;
				this.error_message = "";
				this.events.push(e);
				this.newEvnt = {};
			})
			.catch(err => {
				this.error_message = "Error while adding new squad: " + this.getAxiosErrorMessage(err);
			});
		},
		showAttendies(e, i) {
			let url = `/events/` + e.id + `/participants`;
			if (i != 0) {
				url += `?status=`+this.getEventStatusText(i);
			}
			window.location.href = url;
		},
		deleteEvent(e, i) {
			if(confirm(`Please confirm you really want to delete event ${e.text}, it will be impossible to rollback this operation.`)) {
				axios({
					method: 'DELETE',
					url: '/methods/events/' + e.id,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					this.events.splice(i);
				})
				.catch(err => {
					this.error_message = "Error while removing event " + e.id + ": " + this.getAxiosErrorMessage(err);
				});
			}
		},
		registerForEvent(e, i) {
			if( !e.changingStatus ) {
				e.changingStatus = true;
				axios({
					method: 'POST',
					url: `/methods/events/${e.id}/participants/me`,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					e.status = res.data.status;
					let st = this.getEventStatusText(res.data.status).toLowerCase();
					e[st] = 0 + e[st];
					e[st]++;
					e.changingStatus = false;
				})
				.catch(err => {
					this.error_message = "Error while adding squad member: " + this.getAxiosErrorMessage(err);
					e.changingStatus = false;
				});
			}
		},
		declineEvent(e, i) {
			if( !e.changingStatus ) {
				if(confirm(`Please confirm you really want to decline event ${e.text}.`)) {
					e.changingStatus = true;
					axios({
						method: 'DELETE',
						url: `/methods/events/${e.id}/participants/me`,
						headers: { "X-CSRF-Token": csrfToken },
					})
					.then( res => {
						this.error_message = "";
						let st = this.getEventStatusText(e.status).toLowerCase();
						e[st]--;
						this.events[i].status = 0;
						e.changingStatus = false;
					})
					.catch(err => {
						this.error_message = `Error while removing user ${currentUserId} from event: ` + this.getAxiosErrorMessage(err);
						e.changingStatus = false;
					});
				}
			}
		},
		getParticipantsByStatus(e, i) {
			return e[this.getEventStatusText(i).toLowerCase()];
		},
	},
	mixins: [globalMixin],
}).mount("#app");

