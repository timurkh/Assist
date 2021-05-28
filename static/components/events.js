const AddEventDialog = {
	delimiters: ['[[', ']]'],
	props: {
		windowId: String, 
		title: String,
		squads: Object,
		evnt:{
			type: Object,
			default: () => ({})
		},
	},
	emits: ["submit-form"],
	methods: {
		onSubmit:function() {
			this.$emit('submit-form', this.evnt);
		},
	},
	computed: {
		descriptionNotComplete : function() {
			return this.evnt.date == null || this.evnt.date == "" || this.evnt.text == null || this.evnt.text == "" || this.evnt.squadId == null || this.evnt.squadId == "";
		}
	},
    template:  `
	<div class="modal fade" :id="windowId" tabindex="-1" role="dialog">
		<div class="modal-dialog" role="document">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">[[title]]</h5>
					<button type="button" class="close" data-dismiss="modal" aria-label="Close">
						<span aria-hidden="true">&times;</span>
					</button>
				</div>
				<form>
					<div class="modal-body">
						<div class="row">
							<div class="col-5 form-group">
								<label for="dateTitle">Date</label>
								<input type="date" id="date" class="input-sm form-control" v-model="evnt.date">
							</div>
							<div class="col-7 form-group">
								<label for="dateTitle">Time</label>
								<div class="row">
									<div class="col-5 pr-0">
										<input type="time" id="timeFrom" class="input-sm form-control" v-model="evnt.timeFrom">
									</div>
									<div class="col-2" align="center"> : </div>
									<div class="col-5 pl-0">
										<input type="time" id="timeTo" class="input-sm form-control" v-model="evnt.timeTo">
									</div>
								</div>
							</div>
						</div>
						<div class="form-group">
							<label for="eventText">Description</label>
							<textarea id="eventText" class="form-control" v-model="evnt.text"></textarea>
						</div>
						<div class="form-group">
							<label for="evenSquad">Squad</label>
							<select id="eventSquad" class="form-control" v-model="evnt.squadId">
								<option v-for="squad in squads" :value="squad.id">[[squad.id]]</option>
							</select>
						</div>
					</div>
					<div class="modal-footer">
						<button type="submit" class="btn btn-primary" v-on:click="onSubmit()" data-dismiss="modal" :disabled="descriptionNotComplete">Add</button>
					</div>
				</form>
			</div>
		</div>
	</div>
`
};

export {AddEventDialog};
