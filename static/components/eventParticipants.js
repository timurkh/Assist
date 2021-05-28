const AddParticipantDialog = {
	delimiters: ['[[', ']]'],
	props: {
		windowId: String, 
		candidates: Array,
	},
	data: function() {
		return {
			candidateFilter:"",
			candidatesToJoin:[],
		};
	},
	emits: ["submit-form", "filter-change", "load-more"],
	methods: {
		onSubmit : function() {
			this.$emit('submit-form', this.candidatesToJoin);
		},
		onFilterChange : function() {
			this.$emit('filter-change', this.candidateFilter);
		},
		onScroll (e) {
			if (e.target.scrollTop + e.target.clientHeight >= e.target.scrollHeight) {
				this.$emit('load-more', e);
			}
		}
	},
    template:  `
		<div class="modal fade" :id="windowId" tabindex="-1" role="dialog">
			<div class="modal-dialog" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="participantToChangeStatusModalLabel">Add Participants</h5>
						<button type="button" class="close" data-dismiss="modal" aria-label="Close">
							<span aria-hidden="true">&times;</span>
						</button>
					</div>
					<div class="modal-body">
						<form>
							<div class="form-group">
								<input type="text" id="candidateFilter" class="form-control" v-model="candidateFilter" placeholder="Filter" @input="onFilterChange($event)" @change="onFilterChange($event)">
							</div>
							<div class="form-group">
								<select id="candidatesSelect" multiple v-model="candidatesToJoin" class="form-control" @scroll="onScroll" size="5">
									<option  v-for="(candidate, index) in candidates" :value="candidate">[[candidate.displayName]]</option>
								</select>
							</div>
						</form>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-primary" v-on:click="onSubmit()" data-dismiss="modal">Add</button>
					</div>
				</div>
			</div>
		</div>
`};


export {AddParticipantDialog};
