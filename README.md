### How this application could be used

This app is a kind of 'minimal CRM', might be used to automate various scenarios around groups of people. 

#### Users
One can log into the portal using either email/password, FB identity, Google identity or phone (SMS authorization). After first login, email validation is required to proceed. Having email validated, user gets access to the system. It is possible to enable several auth providers for the same account (i.e. use both phone and FB authorization).

#### Squads
User can create squads. Squad should have unique name which is visible for every user of the system. Other users can join your squad, or you can add so-called *replicants* (user records not bound to particular identity and thus not able to log into the system) to it yourself. If user attempted to join the squad, his status is *Pending Approve*. Squad owner can change user status either to *Member* or *Admin*. Members recieve notifications, can join events, create requests, but are not able to get list of all members, create notes or request queues. Admin has access to all squad members, also admin can change status of other members (but not other admins or owner) and create following entities at the *Squad Details* screen: *Notes*, *Tags*, *Request Queues*, *Events*.

#### Notes
Squad admins & owner can create notes per squad and per squad member. *Squad Notes* are intended to store & share information with all members (non-admins can see notes after they are published). On the contrary, Member Notes are visible only to squad admins. Member does not have access to notes assigned to him in the squad, unless he is squad admin.

#### Tags
Admins can create *Squad Tags*, and assign those tags to members. *Tags* might have values, values are exclusive (only one tag value can be assigned to same member). It is possible to get amount of members with particular tag assigned, and filter members by *tag*. Also *tags* are used to identify request queues approvers and handlers.

#### Request Queues
Squad admins can create *Request Queues*, select tag that identifies users that can approve requests (if left empty, request queue will not have approve stage) and another tag that idenitifes users that should handle them (if left empty, admins are expected to close requests). Approvers and handlers will get browser notifications about new requests (of course if they have permitted them in browser settings).

#### Events
Squad admins can create events, members get notification about new ones and can apply for participation. Admin can approve participation and mark which members did not show-up.

### Technologies, source codes, reliability, costs

This app is written using Go + JS (Vue) + Bootstrap styles and hosted at Google App Engine. Firebase Authentication is used as identity service, Firestore DB is used to store data. Source codes are available [here](https://github.com/timurkh/Assist/).

This system uses Google Application Free tier and might get suspended if monthly quota is exceeded. While quota is quite material (5-10 k pages rendered per day), it is much safer to have the system hosted in your own Google Cloud subscription. Get in touch with author if you want to do it, I will help to deploy.

Also it is possible to reduce Firestore usage, holding more data in memory and thus making fewer read requests (50k requests per day are free as of today).

### About author and why this application was created

Author is available [here](https://www.linkedin.com/in/timur-k/), my CV could be downloaded [here](https://storage.googleapis.com/assist-bucket/Resume-Timur-Khakimyanov.pdf). 
