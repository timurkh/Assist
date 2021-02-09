const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			toc: "",
		};
	},
	mounted:function() {
		this.$nextTick(() => {
			 const matches = document.querySelectorAll(`h4`);
			  matches.forEach(value => {
					const ul = document.createElement('ul');
					const li = document.createElement('li');
					const a = document.createElement('a');
					a.innerHTML = value.textContent;
					a.href = `#${value.id}`;
					li.appendChild(a);
					li.classList.add(this.h2Class);
					ul.appendChild(li);
					toc.appendChild(ul);
			  });
		});
	},
}).mount("#app");
