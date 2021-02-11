const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squadId:squadId,
			notes:[],
			newNote:[],
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/squads/${squadId}`,
		})
		.then(res => {
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve squad details: " + error;
			this.loading = false;
		});
	},
	methods: {
		addNote:function() {
			axios({
				method: 'PUT',
				url: `/methods/squads/${squadId}`,
				data: {
					note : this.newNote,
				},
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.notes.push(Object.assign({}, this.newNote));
			})
			.catch(err => {
				this.error_message = "Error while adding note: " + err;
			});
		}
	},
	mixins: [globalMixin],
}).mount("#app");
