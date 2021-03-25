const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squadId:squadId,
			notes:[],
		};
	},
	created:function() {
		axios.all([
			axios.get(`/methods/squads/${squadId}/notes`),
		])
		.then(axios.spread((notes, tags) => {
			this.notes = notes.data;
			this.loading = false;
		}))
		.catch(errors => {
			this.error_message = "Failed to retrieve squad details: " + this.getAxiosErrorMessage(errors);
			this.loading = false;
		});
	},
	methods: {
		getNoteTitle:function(note) {
			return "[" + (new Date(note.timestamp)).toLocaleDateString() + "] " + note.title;
		},
		editNote:function(note, index) {
			this.noteToEdit = Object.assign({}, note);
			this.noteToEditIndex = index;
			$('#editNoteModal').modal('show');
		},
	},
	mixins: [globalMixin],
}).mount("#app");
