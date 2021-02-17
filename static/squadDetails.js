const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squadId:squadId,
			notes:[],
			tags:[],
			newNote: {},
			newTag: {},
			noteToEdit: {},
			noteToEditIndex: -1,
		};
	},
	created:function() {
		axios.all([
			axios.get(`/methods/squads/${squadId}/notes`),
			axios.get(`/methods/squads/${squadId}/tags`),
		])
		.then(axios.spread((notes, tags) => {
			this.notes = notes.data;
			this.tags = tags.data;
			this.loading = false;
		}))
		.catch(errors => {
			this.error_message = "Failed to retrieve squad details: " + this.getAxiosErrorMessage(errors);
			this.loading = false;
		});
	},
	methods: {
		addTag:function() {

			if(this.newTag.name == "") {
				this.error_message = "Tag name should not be empty.";
				return false;
			}

			let newTag = new Object();
			newTag.name = this.newTag.name;
			newTag.values = new Object();
			if(this.newTag.values != null)
				this.newTag.values.forEach(v => {newTag.values[v] = 0});
			else
				newTag.values["_"] = 0;

			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/tags`,
				data: newTag,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.tags.push(newTag);
			})
			.catch(err => {
				this.error_message = "Error while adding tag: " + this.getAxiosErrorMessage(err);
			});
		},
		addNote:function() {
			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/notes`,
				data: this.newNote,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.newNote.id = res.data.id;
				this.newNote.timestamp = (new Date()).toJSON();
				this.notes.unshift(this.newNote);
				this.newNote = {};
			})
			.catch(err => {
				this.error_message = "Error while adding note: " + this.getAxiosErrorMessage(err);
			});
		},
		toggleNote:function(note, i) {
			note.published = !note.published;
			axios({
				method: 'PUT',
				url: `/methods/squads/${squadId}/notes/${note.id}`,
				data: { published : note.published},
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
			})
			.catch(err => {
				this.error_message = "Error while saving note: " + this.getAxiosErrorMessage(err);
			});
		},
		deleteObject:function(obj, id, index) {
			index = index;
			axios({
				method: 'DELETE',
				url: `/methods/squads/${squadId}/${obj}s/${id}`,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this[`${obj}s`].splice(index, 1);
			})
			.catch(err => {
				this.error_message = `Error while removing ${obj} ${id} from squad: ` + this.getAxiosErrorMessage(err);
			});
		},
		getNoteTitle:function(note) {
			return "[" + (new Date(note.timestamp)).toLocaleDateString() + "] " + note.title;
		},
		editNote:function(note, index) {
			this.noteToEdit = Object.assign({}, note);
			this.noteToEditIndex = index;
			$('#editNoteModal').modal('show');
		},
		saveNote:function() {
			axios({
				method: 'PUT',
				url: `/methods/squads/${squadId}/notes/${this.noteToEdit.id}`,
				data: this.noteToEdit,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				Object.assign(this.notes[this.noteToEditIndex], this.noteToEdit); 
				this.error_message = "";
			})
			.catch(err => {
				this.error_message = "Error while saving note: " + this.getAxiosErrorMessage(err);
			});
		},
	},
	computed: {
		newTagValues : {
			get: function() {
				return this.newTag.values == null? "" : this.newTag.values.join('\n');
			},
			set: function (values) {
				this.newTag.values = values.split('\n');
			}
		},
	}, 
	mixins: [globalMixin],
}).mount("#app");
