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
		getStatusText : function(status) {

			switch(status) {
				case 0:
					return "not going";
				case 1:
					return "applied";
				case 2:
					return "going";
				case 3:
					return "attended";
				case 4:
					return "no-show";
			};
		},
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

		},
		deleteEvent(e, i) {
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
		},
		registerForEvent(e, i) {
			axios({
				method: 'POST',
				url: `/methods/events/${e.id}/participants/me`,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.events[i].status = res.data.status;
			})
			.catch(err => {
				this.error_message = "Error while adding squad member: " + this.getAxiosErrorMessage(err);
			});
		},
		declineEvent(e, i) {
			axios({
				method: 'DELETE',
				url: `/methods/events/${e.id}/participants/${currentUserId}`,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.events[i].status = 0;
			})
			.catch(err => {
				this.error_message = `Error while removing user ${currentUserId} from event:` + this.getAxiosErrorMessage(err);
			});
		},
	},
	mixins: [globalMixin],
}).mount("#app");

