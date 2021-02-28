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
			newEvent:{},
		}
	},
	created:function() {
		this.loading = false;
	},
	methods: {
	},
	mixins: [globalMixin],
}).mount("#app");

