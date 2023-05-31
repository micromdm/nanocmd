/*
Package workflow defines workflow interfaces, types, and primitives.

# Workflows

Workflows are a concept that abstracts away some of the lower-level
tracking of Apple MDM "v1" command responses to better focus on
accomplishing higher-level end goals.

To that end Workflows are, to the larger workflow engine, just MDM
command senders (enqueuers) and receivers for MDM command responses
(Result Reports). However the business of "routing" which MDM commands
are for which purpose including additional metadata is taken care of for
us in a so we can instead concentrate on the actully useful stuff (i.e.
the business logic) of MDM commands.

The interfaces and types defined here are to facilitate that back-end
tracking of MDM commands (by i.e. Command UUID, workflow, and
associating any relevant context (if required) to the workflows so that
when the device responds to MDM command(s) we can restore that context
and "route" the response to the correct workflow for its higher-level
handling.

Workflows are identified by names. By convention these are reverse-DNS
style and are indended to be unique amongst the workflow engine and be
human readable. The workflow names serve as the way to "route" workflow
actions to workflows.

Newly started workflows are given an instance ID. This is just a unique
identifier for tracking or logging. The intent is to associate this ID
to a workflow that has been started and on which devices for logging or
other tracking.

# Steps

Workflows are facilitated by one or more steps. A step is a set of one
or more MDM commands. A newly started workflow enqueues (sends) a
step to one or more devices. A step is completed for an enrollment ID
when all commands in the step are received by a single device — whether
they have an error or not. `NotNow` handling is done for you: a workflow
will only receive a response for an `Acknowledge` or `Error` response to
an MDM command. The step can Timeout — this is when any of the enqueued
commands do not respond within the Timeout given when they were
enqueued. Steps are intended to be sequential for an enrollment ID —
that is a workflow's step completion handler should only enqueue one
step at a time (or none, if the workflow is finished).

Steps are identified by name. There is no convention for these names as
they are workflow specific but they should be human readable as they
will likely be logged and keyed on. It is intended workflows will
identify specific step completions by the name of the step.

# Context

When you enqueue a step you can associate a context value with it. This
context is marshaled into binary (in any way the workflow may like, but
likely to be JSON or a "bare" string). Then, upon step completion, this
same context is unmarshaled and handed back to the workflow's step
completion handler. In this way a workflow can keep track of any data or
metadata between enqueued steps and their responses if you wish. As
mentioned above the step itself also has a name which may preclude the
need for any additional context, but if you need additional context or
data this context is present.

When a workflow is started an initial context can be passed in.
Typically this will be from an API handler that takes data in. The step
name for a newly started workflow is the empty string.

# Process model

No assumptions should be made about the state of the workflow object
receiving method calls. In other words assume the worst: that it's a
shared object (vs, say, newly instantiated in a request context) and
that multiple calls of the methods on the same object will be running
concurrently. Protect any shared resources appropriately (e.g. mutexes
and locking). Even better is to push any saved state into the storage
layers anyway.
*/
package workflow
