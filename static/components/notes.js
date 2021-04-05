const AddNoteDialog = {
	props: {
		windowId: String, 
		title: String,
		note:{
			type: Object,
			default: () => ({})
		},
	},
	emits: ["submit-form"],
	methods: {
		onSubmit:function() {
			this.$emit('submit-form', this.note);
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
							<div class="form-group">
								<label> Title </label>
								<input type="text" class="form-control" v-model="note.title">
							</div>
							<div class="form-group">
								<label> Note </label>
								<textarea class="form-control" v-model="note.text"></textarea>
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
