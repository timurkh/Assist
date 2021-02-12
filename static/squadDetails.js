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
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/squads/${squadId}/tags`,
		})
		.then(res => {
			this.tags = res.data;
			console.log(this.tags);
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve squad details: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		});
	},
	methods: {
		addTag:function() {
			if(this.newTag.name == "") {
				this.error_message = "Tag name should not be empty.";
				return false;
			}

			console.log(this.newTag.values);

			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/tags`,
				data: this.newTag,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.tags.push(Object.assign({}, this.newTag));
			})
			.catch(err => {
				this.error_message = "Error while adding note: " + this.getAxiosErrorMessage(err);
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
				this.notes.push(Object.assign({}, this.newTag));
			})
			.catch(err => {
				this.error_message = "Error while adding note: " + this.getAxiosErrorMessage(err);
			});
		},
		deleteTag:function(tagName, index) {
			index = index;
			axios({
				method: 'DELETE',
				url: `/methods/squads/${squadId}/tags/${tagName}`,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.tags.splice(index, 1);
			})
			.catch(err => {
				this.error_message = `Error while removing tag ${tagName} from squad: ` + this.getAxiosErrorMessage(err);
			});
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
		}
	}, 
	mixins: [globalMixin],
}).mount("#app");
