insert into a_users (
uid, 
last_name, 
first_name, 
email, 
salutation, 
role,
last_login, 
first_login, 
matr_nr, 
academic_title, 
title, 
name_affix,
old_affiliations)

(select 
u.uid as uid,
u.lastname::varchar(255) as last_name,
u.firstname::varchar(255) as first_name,
u.email::varchar(255) as email,

CASE WHEN u.salutation = 'male' THEN 1
     WHEN u.salutation = 'female' THEN 2
     ELSE 0
END as salutation,

CASE WHEN u.urole = 'user' THEN 0
     WHEN u.urole = 'creator' THEN 1
     ELSE 2
END as role,

u.lastlogin as last_login,
now() as first_login,
u.matrnr as matr_nr,
u.academictitle::varchar(127) as academic_title,
u.title::varchar(127) as title,
u.nameaffix::varchar(127) as name_affix,

u.affiliation as old_affiliations

from ldap u);



insert into a_users (
uid, 
last_name, 
first_name, 
email, 
salutation, 
role,
last_login, 
first_login, 
password,
activation_code)

(select 
u.uid as uid,
u.lastname::varchar(255) as last_name,
u.firstname::varchar(255) as first_name,
u.email::varchar(255) as email,

CASE WHEN u.salutation = 'male' THEN 1
     WHEN u.salutation = 'female' THEN 2
     ELSE 0
END as salutation,

CASE WHEN u.urole = 'user' THEN 0
     WHEN u.urole = 'creator' THEN 1
     ELSE 2
END as role,

u.lastlogin as last_login,
to_timestamp(to_char(u.regdate, 'YYYY-MM-DD'), 'YYYY-MM-DD') as first_login,

u.pw::varchar(511) as password,
u.activationcode::varchar(255) as activation_code

from extern u);



insert into studies (user_id, semester, degree_id, course_of_studies_id, touched)

(select 
u.id as user_id, 
s.semester, 
d.id as degree_id, 
c.id as course_of_studies_id,
true

from userstudies s join users u on s.uid = u.uid
join degrees d on d.name  = s.degree
join courses_of_studies c on c.name = s.name)



insert into groups (
name, course_limit, last_editor, last_edited, oldparentid, oldid)

(select
gt.currentpos::varchar(255) as name,

CASE WHEN gt.maxenrollcourses = 0 THEN null
     ELSE 1
END as course_limit,

25, '2018-12-24 06:06:06+00',

gt.parentid as oldparentid,
gt.id as oldid

from groupstree gt
where gt.courseid is null)



insert into a_courses (

title,
creator,
subtitle,
visible,
active,
creation_date,
description, 
fee,
custom_email,
enroll_limit_events,
enrollment_start,
enrollment_end, 
unsubscribe_end, 
expiration_date,
courseid
)

(select

c.title::varchar(511) as title,
(select u.id from users u where u.uid = c.creator) as creator,
c.subtitle::varchar(511) as subtitle,
c.publiclyvisible as visible,
true as active,
'2018-12-24 00:00:00+00' as creation_date,
c.description as description,
round(c.fee::numeric, 2) AS fee,
c.welcomemail as custom_email,
c.enrolllimitevents as enroll_limit_events,
c.enrollstart as enrollment_start,
c.enrollend as enrollment_end,
c.unsubscribeuntil as unsubscribe_end,

CASE WHEN c.inactive and c.expirationdate > now() then '2020-10-10 00:00:00+00'
     WHEN c.inactive and c.expirationdate <= now() THEN c.expirationdate
     ELSE c.expirationdate
END as expiration_date,

c.courseid as courseid

from courses c)



update a_courses ac set parent_id = (

select gr.id 
from groups gr join groupstree gt on gr.oldid = gt.parentid
where gt.courseid = ac.courseid)


select u.first_name, u.last_name, c.title from courses c 
join blacklists b on b.courseid = c.courseid 
join users u on b.uid = u.uid 
where c.courseid = 104;



insert into enrollment_restrictions (
course_id,
minimum_semester,
degree_id,
courses_of_studies_id)

(select

(select c.id from a_courses c where c.courseid = l.courseid) as course_id,
l.minsemester as minimum_semester,
(select d.id from degrees d where d.name = l.degree) as degree_id,
(select s.id from courses_of_studies s where s.name = l.studies) as
courses_of_studies_id

from limitations l)



insert into blacklists 
(user_id, course_id)

( select
(select u.id from users u where b.uid = u.uid) as user_id,
(select c.id from a_courses c where b.courseid = c.courseid) as course_id
from old_blacklists b)


insert into news_feed_category (name, last_editor, last_edited) (
select distinct c.category as name, 25 as last_editor, 
to_timestamp('2020-12-24', 'YYYY-MM-DD') as last_edited
from updatefeed c)

insert into news_feed (
last_editor,
category_id,
content,
last_edited)

(select
25 as last_editor,
(select c.id from news_feed_category c where c.name = u.category)
as category_id,
u.content as content,
u.created as last_edited

from updatefeed u)



insert into faq_category (name, last_editor, last_edited) (
select distinct f.category as name, 25 as last_editor, 
to_timestamp('2020-12-24', 'YYYY-MM-DD') as last_edited
from old_faqs f)

insert into faqs (
last_editor,
category_id,
question,
answer,
last_edited)

(select
25 as last_editor,
(select c.id from faq_category c where c.name = f.category)
as category_id,
f.question::varchar(511) as question,
f.answer::text as answer,
f.created as last_edited

from old_faqs f)



insert into instructors (
user_id,
course_id,
view_matr_nr)

(select

(select u.id from users u where u.uid = l.uid) as user_id,
(select c.id from a_courses c where c.courseid = l.courseid) as course_id,
true as view_matr_nr
from leaders l)



insert into a_events (

course_id,
capacity,
has_waitlist,
title,
enrollment_key,
eventid,
courseid)

(select

(select c.id from a_courses c where c.courseid = e.courseid) as course_id,
e.capacity as capacity,
e.haswaitlist as has_waitlist,
e.description::varchar(255) as title,
e.registrationkey::varchar(511) as enrollment_key,
e.eventid as eventid,
e.courseid as courseid
from events e)



insert into a_meetings (

event_id,
meeting_interval,
weekday,
place,
annotation,
meeting_start,
meeting_end)

(select

(select e.id from a_events e 
where e.eventid = m.eventid and e.courseid = m.courseid) as event_id,

CASE WHEN m.meetinginterval = 'weekly' THEN 1
     WHEN m.meetinginterval = 'evenWeeks' THEN 2
     WHEN m.meetinginterval = 'oddWeeks' THEN 3
     ELSE 0
END as meeting_interval,

CASE WHEN m.dayofweek = 'Monday' THEN 0
     WHEN m.dayofweek = 'Tuesday' THEN 1
     WHEN m.dayofweek = 'Wednesday' THEN 2
     WHEN m.dayofweek = 'Thursday' THEN 3
     WHEN m.dayofweek = 'Friday' THEN 4
     WHEN m.dayofweek = 'Saturday' THEN 5
     ELSE 6
END as weekday,

m.place::varchar(255) as place,
m.annotation::varchar(255) as annotation,
m.starts as meeting_start,
m.ends as meeting_end

from meetings m)



insert into enrolled (
user_id,
event_id,
status,
time_of_enrollment
)

(select
(select u.id from users u where u.uid = en.uid) as user_id,
(select e.id from a_events e where e.eventid = en.eventid and e.courseid =
en.courseid) as event_id,

CASE WHEN en.ustatus = 'enrolled' THEN 0
     WHEN en.ustatus = 'on waitlist' THEN 1
     WHEN en.ustatus = 'awaiting payment' THEN 2
    WHEN en.ustatus = 'paid' THEN 3
     ELSE 4
END as status,

en.enrolldate as time_of_enrollment

from old_enrolled en)



insert into unsubscribed (
user_id,
event_id)

(select

(select u.id from users u where u.uid = un.uid) as user_id,
(select e.id from a_events e where e.eventid = un.eventid
and e.courseid = un.courseid) as event_id

from old_unsubscribed un)
