const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			squadId:squadId,
			squad_members:[],
			squad_owner:null,
			statusToSet:0,
			changeStatusMember_index: -1,
			changeStatusMember: [],
			replicant: [],
		};
	},
	created:function() {
		axios({
			method: 'GET',
			url: `/methods/squads/${squadId}/members`,
		})
		.then(res => {
			this.squad_members = res.data['Members']; 
			this.squad_owner = res.data['Owner'];
			this.loading = false;
		})
		.catch(error => {
			this.error_message = "Failed to retrieve list of squad members: " + this.getAxiosErrorMessage(error);
			this.loading = false;
		});
	},
	methods: {
		changeStatus:function(index, member) {
			this.changeStatusMember_index = index;
			this.changeStatusMember = member;
			this.statusToSet = member.status;
			$('#changeMemberStatusModal').modal('show')
		},
		setMemberStatus:function() {
			axios({
				method: 'PUT',
				url: `/methods/squads/${squadId}/members/${this.changeStatusMember.id}`,
				data: {
					Status: this.statusToSet,
				},
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.squad_members[this.changeStatusMember_index].status = this.statusToSet;
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
		addMember:function() {
			axios({
				method: 'POST',
				url: `/methods/squads/${squadId}/members`,
				data: {
					displayName : this.replicant.displayName,
					email : this.replicant.email,
					phoneNumber : this.replicant.phoneNumber,
					replicant: true,
				},
				headers: { "X-CSRF-Token": csrfToken },
			})
			.then( res => {
				this.error_message = "";
				this.replicant.id = res.data.ReplicantId;
				this.replicant.status = 1; //member
				this.replicant.replicant = true;
				this.squad_members.push(Object.assign({}, this.replicant));
			})
			.catch(err => {
				this.error_message = "Error while adding squad member: " + this.getAxiosErrorMessage(err);
			});
		}
	},
	mixins: [globalMixin],
}).mount("#app");
