### How this application could be used

This app is a kind of 'minimal CRM for personal usage'. While developing it, I've taken scenarios from my shooting club as a model: we have several hundreds members and events with pre registration. GMail groups + FB + Excel mix was sufficient to handle all scenarios for years, but not too convinient.

##### Users
One can log into the portal using either email/password, FB identity, Google identity or phone (SMS authorization). After first login, email validation is required to proceed. Having email validated, user gets access to the system. It is possible to enable several auth providers for the same account (i.e. use both phone and FB authorization).

##### Squads
User can create squads. Squad is visible for every user of the system. Other users can join your squad, or you can add so-called "replicants" to it yourself. If user attempted to join the squad, his status is "Pending Approve". Squad owner can change user status either to Member or Admin. Members recieve notifications, can join events, but are not able to communicate squad or get list of all members. Admin has access to all squad members, also admin can change status of other members (but not admin ones or owner).

##### Notes
Squad admins & owner can create notes per squad and per squad member. Squad Notes are intended to store & share information with the whole squad and are visible to all squad members. On the contrary, Member Notes are visible only to squad admins. Member does not have access to notes assigned to him in the squad, unless he is squad admin.

##### Tags
Admins can create Squad Tags, and assign those tags to members. Tags might have values, values are exclusive (only one tag value can be assigned to same member).

### Technologies, source codes, reliability, costs

This app is written using Go + JS (Vue) + Bootstrap styles and hosted at Google App Engine. Firebase Authentication is used as identity service, Firestore DB is used to store data. Source codes are available [here](https://github.com/timurkh/Assist/).

This system uses Google Application Free tier and might get suspended if monthly quota is exceeded. While quota is quite material (5-10 k pages rendered per day), it is much safer to have the system hosted in your own Google Cloud subscription. Get in touch with author if you want to do it, I will help to deploy.

Also it is possible to reduce Firestore usage, holding more data in memory and thus making fewer read requests (50k requests per day are free as of today).

### About author and why this application was created

~~Because I can~~ As a part of technology refresh during my career break after serving 5 years as Executive Director in the world's largest provider of information technology products and services, I decided to implement few apps (web & android) automating certain scenarios which I face in my day-to-day life, mostly related to people, places and events.

Author is available [here](https://www.linkedin.com/in/timur-k/), my CV could be downloaded [here](https://storage.googleapis.com/assist-bucket/Resume-Timur-Khakimyanov.pdf). 
