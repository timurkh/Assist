const AddMemberDialog = {
	delimiters: ['[[', ']]'],
	props: {
		windowId: String, 
	},
	data: function() {
		return {
			replicant:{},
		};
	},
	emits: ["submit-form"],
	methods: {
		onSubmit : function() {
			this.$emit('submit-form', this.replicant);
			this.replicant = {};
		},
	},
    template:  `
		<div class="modal fade" :id="windowId" tabindex="-1" role="dialog">
			<div class="modal-dialog" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="changeMemberStatusModalLabel">Add Member <i class="fas fa-robot"></i></h5>
						<button type="button" class="close" data-dismiss="modal" aria-label="Close">
							<span aria-hidden="true">&times;</span>
						</button>
					</div>
					<div class="modal-body">
						<form>
							<div class="form-group">

								<label for="displayName">Name</label>
								<input type="text" id="displayName" class="form-control" v-model="replicant.displayName">
								<label for="Email">Email</label>
								<input type="text" id="Email" class="form-control" v-model="replicant.email">
								<label for="PhoneNumber">Phone</label>
								<input type="text" id="PhoneNumber" class="form-control" v-model="replicant.phoneNumber">
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


const ChangeStatusDialog = {
	delimiters: ['[[', ']]'],
	props: {
		windowId: String, 
		member: Object, 
	},
	emits: ["submit-form"],
	methods: {
		onSubmit : function() {
			this.$emit('submit-form', this.member.status);
		},
	},
	mixins: [globalMixin],
    template:  `
		<div class="modal fade" id="changeMemberStatusModal" tabindex="-1" role="dialog">
			<div class="modal-dialog" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="changeMemberStatusModalLabel">[[member.displayName]] Status</h5>
						<button type="button" class="close" data-dismiss="modal" aria-label="Close">
							<span aria-hidden="true">&times;</span>
						</button>
					</div>
					<div class="modal-body">
						<form>
							<div class="form-group">
								<select v-model="member.status" class="form-control" size="3">
									<option  v-for="i in 3" :value="i-1">[[getStatusText(i-1)]]</option>
								</select>
							</div>
						</form>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-primary" v-on:click="onSubmit()" data-dismiss="modal">Set Selected Status</button>
					</div>
				</div>
			</div>
		</div>
`
};

const AddTagDialog = {
	delimiters: ['[[', ']]'],
	props: {
		windowId: String, 
		tags: Array,
		member: Object, 
	},
	data : function() {
		return {
			tagToSet:this.tags && this.tags.length > 0 ? Object.assign({}, this.tags[0]) : {},
			tagToSetValue:"",
		};
	},
	emits: ["submit-form"],
	methods: {
		onSubmit : function() {
			if(this.tagToSet == null)
				return;

			let tag = new Object();
			tag.Name = this.tagToSet.name;
			if(this.getTagHasValues(this.tagToSet))
				tag.Value = this.tagToSetValue;

			this.$emit('submit-form', tag);
		},
	},
	computed: {
		newTagValue: {
			get: function() {
				if (this.tagToSetValue.length == 0) {
					this.tagToSetValue = Object.keys(this.tagToSet.values)[0];
				}

				return this.tagToSetValue;
			},
			set: function (value) {
				this.tagToSetValue = value;
			}
		},
	}, 
	mixins: [globalMixin],
    template:  `
		<div class="modal fade" :id="windowId" tabindex="-1" role="dialog">
			<div class="modal-dialog" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="addTagModalLabel">Tag [[member.displayName]]</h5>
						<button type="button" class="close" data-dismiss="modal" aria-label="Close">
							<span aria-hidden="true">&times;</span>
						</button>
					</div>
					<div class="modal-body pb-0">
						<form>
							<div class="form-group">
								<label for="tagToSet">Choose Tag</label>
								<select id="tagToSet"  v-model="tagToSet" class="form-control">
									<option  v-for="tag in tags" :value="tag">[[tag.name]]</option>
								</select>

								<label for="tagToSetVal" v-if="getTagHasValues(tagToSet)" class="mt-3">Choose Tag Value</label>
								<select id="tagToSetVal" v-if="getTagHasValues(tagToSet)" v-model="newTagValue" class="form-control">
									<option  v-for="(c, v) in tagToSet.values" :value="v">[[v]]</option>
								</select>
							</div>
						</form>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-primary" v-on:click="onSubmit()" data-dismiss="modal">Set</button>
					</div>
				</div>
			</div>
		</div>
		`
};

const ShowNoteDialog = {
	delimiters: ['[[', ']]'],
	props: {
		windowId: String, 
		note: Object, 
	},
	data : function () {
		return {
			editing: false,
		};
	},
	emits: ["submit-form", "delete"],
	methods: {
		onSubmit : function() {
			if(this.editing) {
				this.$emit('submit-form', this.note);
				$(`#${this.windowId}`).modal('hide')
			}
			else
				this.editing = true;
		},
		onDelete : function() {
			this.$emit('delete', this.note);
			$(`#${this.windowId}`).modal('hide')
		},
		getTextAreaHeight:function() {
			if(this.note.text) {
				return this.note.text.split('\n').length;
			}

			return 5;
		},
		getSaveButtonText:function() {
			return this.editing? "Save note" : "Edit note";
		},
	},
    template:  `
<div class="modal fade" :id="windowId" tabindex="-1" role="dialog" aria-labelledby="modalLongTitle" aria-hidden="true">
	  <div class="modal-dialog modal-lg" role="document">
	    <div class="modal-content">
	      <div class="modal-header">
	        <h5 class="modal-title" id="modalLongTitle">[[note.title]]</h5>
	        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
	          <span aria-hidden="true">&times;</span>
	        </button>
	      </div>
	      <div class="modal-body">
		  <textarea v-model="note.text" :disabled="!editing" class="form-control" :rows="getTextAreaHeight()">
			</textarea>
	      </div>
	      <div class="modal-footer">
	        <button type="button" class="btn btn-secondary" data-dismiss="modal">Close</button>
	        <button type="button" class="btn btn-primary" v-on:click="onSubmit()">[[getSaveButtonText()]]</button>
	        <button type="button" class="btn btn-danger" v-on:click="onDelete()">Delete note</button>
	      </div>
	    </div>
	  </div>
	</div>
`
};
