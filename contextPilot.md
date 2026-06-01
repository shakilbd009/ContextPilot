1. Project Overview

The project is a meeting intelligence system that helps users prepare for meetings by remembering what happened in previous conversations and surfacing the most relevant information before the next call.

The product should ensure that users do not enter meetings without context. It should help them understand previous discussions, unresolved issues, action items, stakeholder concerns, risks, and recommended focus areas before the meeting starts.

This is not a generic meeting notes tool. The product is focused on meeting continuity, memory, and pre-call preparation.

Core promise:

Never walk into a meeting cold again.

2. Business Problem

Professionals often attend many meetings across different projects, clients, teams, and stakeholders. Important details are scattered across meeting notes, transcripts, emails, documents, chats, and memory.

As a result:

Users forget what was discussed in previous meetings.
Action items are missed or delayed.
Decisions are not clearly remembered.
Stakeholder concerns are forgotten.
Follow-up meetings start without proper context.
Users waste time reviewing old notes manually.
Meetings repeat the same discussions instead of moving forward.
Important questions are not asked at the right time.

The business need is to create a system that acts like a reliable memory layer for meetings and prepares the user before each future conversation.

3. Business Objective

The objective is to give users a clear, useful, and timely briefing before each meeting so they can perform better, follow up properly, and drive better outcomes.

The product should help users:

remember what happened before
know what is still unresolved
understand who owns what
identify decisions already made
recognize risks and blockers
prepare better questions
focus the next meeting on the right outcome
build stronger continuity across conversations
4. Target Users

The system is intended for professionals who manage recurring or high-context conversations.

Primary users include:

founders
executives
consultants
client-facing engineers
project managers
product managers
sales professionals
customer success managers
recruiters
legal or compliance professionals
team leads
account managers

The strongest early use case is for users who have repeated meetings with the same clients, teams, projects, or stakeholders.

5. Product Vision

The product should behave like a personal meeting memory assistant.

Before a call, the user should be able to quickly understand:

what happened last time
what was promised
what was decided
what is still open
what risks exist
what each important person cares about
what should be discussed next
what outcome should be achieved

The long-term vision is to create an intelligent business memory layer that connects meetings, people, tasks, decisions, risks, and follow-ups.

6. Core User Experience

The product should support three key moments.

6.1 Before the Meeting

Before a meeting starts, the user receives a concise briefing.

The briefing should explain:

why the meeting matters
what happened in previous related meetings
what unresolved items remain
who owes what
what risks or blockers exist
what questions should be asked
what the recommended meeting objective is
what the user should avoid forgetting

This is the most important part of the product.

6.2 After the Meeting

After a meeting, the system should capture and organize what happened.

It should identify:

meeting summary
decisions
action items
owners
deadlines
risks
blockers
unresolved questions
stakeholder concerns
next recommended focus

This information becomes memory for future meetings.

6.3 Across Multiple Meetings

The system should connect related meetings over time.

It should understand when meetings are connected by:

same attendees
same client
same project
same topic
same recurring meeting series
similar meeting titles
related action items or risks

This allows the system to build continuity instead of treating every meeting as isolated.

7. Expected Features
7.1 Meeting Capture

The system should allow users to capture or import meeting information.

Expected capabilities:

create or identify a meeting
capture meeting title
capture date and time
capture attendees
capture organizer
capture meeting description
capture transcript or notes
associate meeting with a project, client, or topic where possible
7.2 Meeting Summary

The system should generate a clear summary of each meeting.

The summary should include:

main topics discussed
key outcomes
important concerns raised
unresolved discussion points
next steps

The summary should be short enough to review quickly but detailed enough to restore context.

7.3 Decision Tracking

The system should identify and track decisions made during meetings.

Each decision should include:

what was decided
who made or agreed to the decision
when it was decided
related meeting
supporting context where available

The system should avoid confusing general discussion with actual decisions.

7.4 Action Item Tracking

The system should identify action items from meetings.

Each action item should include:

task description
owner
due date, if mentioned
status
related meeting
source context where available

The system should help users see what is still open before the next meeting.

7.5 Risk and Blocker Tracking

The system should detect risks, blockers, and unresolved dependencies.

Examples:

approval is delayed
budget is not confirmed
a stakeholder has concerns
a deadline is at risk
technical dependency is unresolved
security or compliance review is pending

Each risk should include:

description
severity if known
status
related meeting
owner or responsible party if known
7.6 Open Question Tracking

The system should identify questions that were raised but not answered.

Examples:

Who owns final approval?
Is the timeline still realistic?
Has the security review been completed?
Has the budget been approved?
What happens if the deadline slips?

These open questions should be carried into the next briefing when relevant.

7.7 Stakeholder Memory

The system should remember useful context about people involved in meetings.

Examples:

a stakeholder is concerned about security
a stakeholder asked for written documentation
a stakeholder is waiting on budget approval
a stakeholder appears to be a decision maker
a stakeholder promised to follow up

The system should only include stakeholder notes that are relevant, professional, and grounded in meeting context.

It should not make unsupported personal assumptions.

7.8 Related Meeting Detection

The system should identify when an upcoming meeting is related to previous meetings.

Relatedness may be based on:

same attendees
same client
same project
same company
similar meeting title
similar discussion topic
recurring meeting pattern
open action items linked to the meeting

This allows the system to bring forward the right memory before the next call.

7.9 Pre-Call Briefing

The system should generate a useful briefing before an upcoming meeting.

The briefing should include:

meeting title
meeting objective
previous meeting context
important decisions already made
open action items
unresolved risks or blockers
open questions
stakeholder notes
suggested questions to ask
suggested agenda
recommended focus
suggested opening statement, where useful

This is the flagship feature.

7.10 “What Changed Since Last Time?” View

The system should show what changed since the previous related meeting.

Examples:

an action item was completed
a risk is still open
a stakeholder replied with new information
a decision has changed
a new blocker appeared
a new attendee has joined the next meeting

This helps the user avoid reviewing everything manually.

7.11 Promise and Commitment Tracking

The system should track commitments made by the user and by others.

Examples:

“I will send the architecture diagram by Friday.”
“Sarah will review the checklist.”
“David will confirm budget approval.”
“The team will decide on rollout date next week.”

Before the next meeting, the user should know:

what they promised
what others promised
what is completed
what is still pending
7.12 Suggested Questions

The system should recommend questions for the user to ask in the next meeting.

Questions should be based on:

unresolved action items
prior decisions
open risks
stakeholder concerns
missing approvals
timeline uncertainty
unclear ownership

Example questions:

Has the security checklist been reviewed?
Can we confirm budget approval today?
Is the pilot date still realistic?
Who owns the final go/no-go decision?
What is blocking the next step?
7.13 Suggested Agenda

The system should recommend a focused agenda for the upcoming meeting.

Example:

Confirm security review status.
Confirm budget approval.
Review pilot timeline.
Resolve open blockers.
Assign owners and dates.

The agenda should help the meeting move forward instead of repeating old discussions.

7.14 Recommended Meeting Objective

The system should recommend the main objective for the upcoming meeting.

Examples:

confirm approval status
resolve blocker
align on timeline
secure decision
clarify ownership
close open action items
prepare next phase

The objective should be direct and outcome-focused.

7.15 Follow-Up Summary

After a meeting, the system should generate a follow-up summary.

The follow-up summary should include:

what was discussed
what was decided
who owns what
next steps
due dates
unresolved issues
recommended next meeting focus

This can later be used to draft follow-up emails or updates.

7.16 Dashboard

The product should include a dashboard where users can see:

upcoming meetings
meetings that need preparation
generated briefings
open action items
unresolved risks
recent meeting memories
important follow-ups
meetings missing transcripts or notes

The dashboard should help users quickly understand where their attention is needed.

7.17 Meeting Detail View

Each meeting should have a detail view showing:

meeting title
date and time
attendees
meeting notes or transcript
summary
decisions
action items
risks
open questions
stakeholder notes
related future meetings
generated briefing, if applicable
7.18 Manual Meeting Input

The first release should allow users to manually enter meeting information.

Users should be able to:

create a meeting
add attendees
paste notes or transcript
process the meeting into memory
create a future meeting
generate a briefing from previous context

This allows the product to prove value before full platform integrations are added.

7.19 Future Meeting Platform Integrations

The product should eventually integrate with major meeting platforms.

Expected future integrations:

Microsoft Teams
Google Meet
Zoom
calendar systems
email systems
document systems
CRM systems
project management tools

The system should eventually be able to automatically collect meeting information where permissions allow.

8. Initial Release Requirements

The first release should prove the core value manually.

The user should be able to:

Create a past meeting.
Add attendees.
Paste meeting transcript or notes.
Generate meeting memory.
See summary, decisions, action items, risks, and open questions.
Create an upcoming related meeting.
Generate a pre-call briefing.
View what to focus on before the next meeting.

The first release does not need live meeting capture, meeting bots, real-time transcription, or full platform integrations.

The goal is to prove that previous meeting context can be turned into useful preparation for the next meeting.

9. Future Release Requirements

Future releases may include:

automatic calendar sync
automatic transcript import
Teams integration
Google Meet integration
Zoom integration
email briefing delivery
Teams/Slack notification delivery
CRM integration
Jira/Linear integration
document retrieval
live meeting assistant
browser extension
stakeholder relationship map
contradiction detection
automatic follow-up email drafts
automatic task creation
organization-wide team memory
enterprise admin controls
10. Business Rules

The system should follow these business rules:

Meeting memory must be tied to a specific user or organization.
Users should only see meeting data they are allowed to access.
The system should not invent facts that are not supported by meeting content.
Decisions should be clearly separated from general discussion.
Action items should include owners and due dates where available.
Risks should remain visible until resolved or dismissed.
Open questions should carry forward into future briefings.
Stakeholder notes should be professional and relevant.
The system should prioritize useful preparation over long summaries.
Briefings should be concise, actionable, and focused on the next meeting.
11. Success Criteria

The project is successful if users can quickly understand what they need to know before a meeting without manually reviewing old notes.

Success can be measured by:

users viewing briefings before meetings
users completing more follow-ups
fewer missed action items
fewer repeated discussions
faster meeting preparation
better decision tracking
users reporting that briefings are useful
users relying on the system for recurring meetings
12. Example User Flow

A user has a client meeting called Acme Security Review.

After the meeting, the system records:

the client agreed to a phased rollout
security checklist review is still pending
Sarah needs to review the checklist
David has not confirmed budget approval
the pilot date may slip if security approval is delayed
the user promised to send an updated architecture diagram

Before the next meeting, Acme Pilot Follow-Up, the system generates:

Pre-Call Briefing: Acme Pilot Follow-Up

Recommended Objective:
Confirm security review status, budget approval, and pilot readiness.

Previous Context:
Last time, the team agreed to move forward with a phased rollout.
The main blocker was the security checklist review.
Budget approval was not explicitly confirmed.

Open Items:
- Sarah needs to complete security checklist review.
- David needs to confirm budget approval.
- You promised to send the updated architecture diagram.

Risks:
- Pilot date may slip if security review is delayed.
- Budget owner has not clearly approved the pilot.

Suggested Questions:
- Has the security checklist been reviewed?
- Can we confirm budget approval today?
- Is the pilot date still realistic?
- Who owns the final go/no-go decision?

Suggested Agenda:
1. Confirm security status.
2. Confirm budget status.
3. Review pilot timeline.
4. Assign owners and next steps.
13. Expected Outcome

The expected outcome is a product that turns meetings into reusable business memory.

Instead of simply storing meeting notes, the system should continuously help the user prepare, follow up, and move conversations forward.

The product should become the user’s trusted pre-call intelligence layer.