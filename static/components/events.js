const AddEventDialog = {
	props: {
		windowId: String, 
		title: String,
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
    template:  `
	<div class="modal fade" :id="windowId" tabindex="-1" role="dialog">
		<div class="modal-dialog" role="document">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">{{title}}</h5>
					<button type="button" class="close" data-dismiss="modal" aria-label="Close">
						<span aria-hidden="true">&times;</span>
					</button>
				</div>
				<form>
					<div class="modal-body">
						<div class="row">
							<div class="col form-group">
								<label for="dateTitle">Date</label>
								<input type="date" id="date" class="input-sm form-control" v-model="evnt.date">
							</div>
							<div class="col form-group">
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
							<label for="eventText">Event</label>
							<textarea id="eventText" class="form-control" v-model="evnt.text"></textarea>
						</div>
					</div>
					<div class="modal-footer">
						<button type="submit" class="btn btn-primary" v-on:click="onSubmit()" data-dismiss="modal">Add</button>
					</div>
				</form>
			</div>
		</div>
	</div>
`
};
