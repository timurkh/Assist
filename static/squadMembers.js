const { createApp } = Vue

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
			squad_owner:null,
			changeMember: [],
			tags:[],
			note:{},
		};
	},
	created:function() {
		axios.all([
			axios.get(`/methods/squads/${squadId}/members`),
			axios.get(`/methods/squads/${squadId}/tags`),
		])
		.then(axios.spread((members, tags) => {
			this.squad_members = members.data['Members']; 
			this.squad_owner = members.data['Owner'];
			this.tags = tags.data;
			this.loading = false;
		}))
		.catch(error => {
			this.error_message = "Failed to retrieve squad members and tags: " + this.getAxiosErrorMessage(error);
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
		removeMember:function(userId, index) {
			index = index;
			axios({
				method: 'DELETE',
				url: `/methods/squads/${squadId}/members/${userId}`,
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.squad_members.splice(index, 1);
			})
			.catch(err => {
				this.error_message = `Error while removing user ${userId} from squad:` + this.getAxiosErrorMessage(err);
			});
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
			delete notes[note.title];
			this.saveNotes(note.member_id, notes);
		},
	},
	mixins: [globalMixin],
}).mount("#app");
