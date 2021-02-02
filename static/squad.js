const app = new Vue({
	el:'#app',
	delimiters: ['[[', ']]'],
	data:{
		loading:true,
		error_message:"",
		squad_members:[],
		squad_owner:null,
		statusToSet:0,
		changeStatusMember_index: -1,
		changeStatusMember: [],
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
			this.error_message = "Failed to retrieve list of squad members: " + error;
			this.loading = false;
		})
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
				}
			})
			.then( res => {
				this.error_message = "";
				this.squad_members[this.changeStatusMember_index].status = this.statusToSet;
			})
			.catch(err => {
				this.error_message = "Error while changing member status: " + err;
			});
		},
		removeMember:function(userId, index) {
			index = index;
			axios({
				method: 'delete',
				url: `/methods/squads/${squadId}/members/${userId}`,
			})
			.then( res => {
				this.error_message = "";
				this.squad_members.splice(index, 1);
			})
			.catch(err => {
				this.error_message = `Error while removing user ${userId} from squad:` + err;
			});
		},
	},
})
