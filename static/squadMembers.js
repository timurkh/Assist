const app = createApp( {
	delimiters: ['[[', ']]'],
	components: {
		'add-note-dialog' : AddNoteDialog,
		'add-member-dialog' : AddMemberDialog,
		'change-status-dialog' : ChangeStatusDialog,
		'add-tag-dialog' : AddTagDialog,
		'show-note-dialog' : ShowNoteDialog,
	},
	data:function(){
		return {
			loading:true,
			error_message:"",
			squadId:squadId,
			squad_members:[],
			changeMember: [],
			tags:[],
			note:{},
			getting_more:false,
			filter:{ },
			moreRecordsAvailable: false,
		};
	},
	created:function() {
		let uri = window.location.search.substring(1); 
		let params = new URLSearchParams(uri);
		this.filter.status = params.get("status");
		this.filter.tag = params.get("tag");

		axios.all([
			axios.get(`/methods/squads/${squadId}/members`, {params : this.filter}),
			axios.get(`/methods/squads/${squadId}/tags`),
		])
		.then(axios.spread((members, tags) => {
			this.moreRecordsAvailable = members.data.length == 10;
			this.squad_members = members.data; 
			this.tags = tags.data;
			this.loading = false;
		}))
		.catch(err => {
			this.error_message = "Failed to retrieve squad members and tags: " + this.getAxiosErrorMessage(err);
			this.loading = false;
		});
	},
	methods: {

		changeStatus:function(member, index) {
			this.changeMember = member;
			this.changeMember.index = index;
			$('#changeMemberStatusModal').modal('show')
		},
		tagMember:function(member, index) {
			this.changeMember = member;
			this.changeMember.index = index;
			$('#addTagModal').modal('show')
		},
		setMemberTag:function(tag) {
			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/members/${this.changeMember.id}/tags`,
				data: { Tag: tag}, 
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.squad_members[this.changeMember.index].tags = res.data.tags;
			})
			.catch(err => {
				this.error_message = "Error while tagging member: " + this.getAxiosErrorMessage(err);
			});
		},
		deleteMemberTag:function(member, tag, tagIndex) {
			if(confirm(`Please confirm you really want to delete tag ${tag} from user ${member.displayName}`)) {
				axios({
					method: 'DELETE',
					url: `/methods/squads/${squadId}/members/${member.id}/tags/${tag}`,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					member.tags = res.data.tags;
				})
				.catch(err => {
					this.error_message = "Error while updating member tags: " + this.getAxiosErrorMessage(err);
				});
			}
		},
		setMemberStatus:function(status) {
			axios({
				method: 'PATCH',
				url: `/methods/squads/${squadId}/members/${this.changeMember.id}`,
				data: { status: status, },
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.squad_members[this.changeMember.index].status = status;
			})
			.catch(err => {
				this.error_message = "Error while changing member status: " + this.getAxiosErrorMessage(err);
			});
		},
		removeMember:function(member, index) {
			if(confirm(`Please confirm you really want to delete user ${member.displayName} from squad ${squadId}`)) {
				index = index;
				axios({
					method: 'DELETE',
					url: `/methods/squads/${squadId}/members/${member.id}`,
					headers: { "X-CSRF-Token": csrfToken },
				})
				.then( res => {
					this.error_message = "";
					this.squad_members.splice(index, 1);
				})
				.catch(err => {
					this.error_message = `Error while removing user ${member.id} from squad:` + this.getAxiosErrorMessage(err);
				});
			}
		},
		addMember:function(replicant) {
			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/members`,
				data: {
					displayName : replicant.displayName,
					email : replicant.email,
					phoneNumber : replicant.phoneNumber,
					replicant: true,
				},
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				replicant.id = res.data.ReplicantId;
				replicant.status = 1; //member
				replicant.replicant = true;
				this.squad_members.push(replicant);
			})
			.catch(err => {
				this.error_message = "Error while adding squad member: " + this.getAxiosErrorMessage(err);
			});
		},
		addNote : function (member, index) {
			this.note = new Object();
			this.note.dialog_title = `Add note about ${member.displayName}`;
			this.note.member_id = member.id;
			this.note.member_index = index;
			$('#addNoteModal').modal('show')
		},
		saveNotes:function(userId, notes) {
			method = 'PATCH';
			url = `/methods/squads/${squadId}/members/${userId}`;
			axios({
				method: method,
				url: url,
				data: { notes : notes },
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
			})
			.catch(err => {
				this.error_message = "Error while saving note: " + this.getAxiosErrorMessage(err);
			});
		},
		submitNote : function(note) {
			let member = this.squad_members[note.member_index];
			if (member.notes == null) {
				member.notes = {};
			}
			let notes = member.notes;
			notes[note.title] = note.text;
			this.saveNotes(note.member_id, notes);
		},
		showNote:function (title, text, member, index) {
			this.note = {};
			this.note.title = title;
			this.note.text = text;
			this.note.member_id = member.id;
			this.note.member_index = index;
			this.$refs.showNoteDialogRef.editing = false;
			$('#showNoteModal').modal('show')
		},
		deleteNote:function(note) {
			let member = this.squad_members[note.member_index];
			let notes = member.notes;
			if(confirm(`Please confirm you really want to delete note ${note.title} from user ${member.displayName}`)) {
				delete notes[note.title];
				this.saveNotes(note.member_id, notes);
			}
		},
		getMore:function() {
			this.getting_more = true;
			let lastMember = this.squad_members[this.squad_members.length-1];
			axios({
				method: 'GET',
				url: `/methods/squads/${squadId}/members?from=${lastMember.timestamp}`,
				params: this.filter,
			})
			.then(res => {
				this.moreRecordsAvailable = res.data.length == 10;
				this.squad_members =  [...this.squad_members, ...res.data]; 
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
				url: `/methods/squads/${squadId}/members`,
				params: this.filter, 
			})
			.then( res => {
				this.error_message = "";
				this.moreRecordsAvailable = res.data.length == 10;
				this.squad_members = res.data; 
				this.loading = false;
			})
			.catch(err => {
				this.error_message = "Failed to retrieve squad members: " + this.getAxiosErrorMessage(err);
				this.loading = false;
			});
		},
	},
	mixins: [globalMixin],
}).mount("#app");
