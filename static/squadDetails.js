import {AddNoteDialog} from "/static/components/notes.js";

const app = createApp( {
	delimiters: ['[[', ']]'],
	components: {
		'note-dialog' : AddNoteDialog,
	},
	data:function(){
		return {
			loading:true,
			error_message:"",
			squadId:squadId,
			squad:{},
			notes:[],
			tags:[],
			newTag: {},
			noteToEdit:{},
			noteNew:{},
			newQueue:{},
			queues:[],
		};
	},
	created:function() {
		axios.all([
			axios.get(`/methods/squads/${squadId}`),
			axios.get(`/methods/squads/${squadId}/notes`),
			axios.get(`/methods/squads/${squadId}/tags`),
			axios.get(`/methods/squads/${squadId}/queues`),
		])
		.then(axios.spread((squad,notes, tags, queues) => {
			this.squad = squad.data;
			this.notes = notes.data;
			this.tags = tags.data;
			this.queues = queues.data;
			this.loading = false;
		}))
		.catch(errors => {
			this.error_message = "Failed to retrieve squad details: " + this.getAxiosErrorMessage(errors);
			this.loading = false;
		});
	},
	methods: {
		getWaitingApproveRequestsCount:function() {
			return this.queues.reduce((a, c)  => a + c.requestsWaitingApprove, 0);
		},
		getProcessingRequestsCount:function() {
			return this.queues.reduce((a, c)  => a + c.requestsProcessing, 0);
		},
		addQueue:function() {
			this.newQueue.id = this.newQueue.id.trim();
			this.newQueue.requestsWaitingApprove = 0;
			this.newQueue.requestsProcessing = 0;

			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/queues`,
				data: this.newQueue,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.queues.push(Object.assign({}, this.newQueue));
			})
			.catch(err => {
				this.error_message = "Error while adding queue: " + this.getAxiosErrorMessage(err);
			});
		},
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
		addNote:function(note) {
			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/notes`,
				data: note,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				note.id = res.data.id;
				note.timestamp = (new Date()).toJSON();
				this.notes.unshift(note);
				this.noteNew = {};
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
			if(confirm(`Please confirm you really want to delete ${obj} from squad ${id}`)) {
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
			}
		},
		getNoteTitle:function(note) {
			return "[" + (new Date(note.timestamp)).toLocaleDateString() + "] " + note.title;
		},
		editNote:function(note, index) {
			Object.assign(this.noteToEdit, note);
			this.noteToEdit.index = index;
			$('#editNoteModal').modal('show');
		},
		saveNote:function(note) {
			axios({
				method: 'PUT',
				url: `/methods/squads/${squadId}/notes/${note.id}`,
				data: note,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				Object.assign(this.notes[note.index], note); 
				this.error_message = "";
			})
			.catch(err => {
				this.error_message = "Error while saving note: " + this.getAxiosErrorMessage(err);
			});
		},
		showTag:function(tag) {
			window.location.href = "/squads/" + encodeURI(squadId) + "/members?tag=" + encodeURI(tag);
		}
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
