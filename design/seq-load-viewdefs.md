# Sequence: Load Viewdefs

**Source Spec:** viewdefs.md
**Use Case:** Loading and validating viewdefs from variable 1 on frontend

## Participants

- FrontendApp: Frontend application
- ViewdefStore: Viewdef storage and validation
- View: Pending views waiting for viewdefs
- ProtocolHandler: Error reporting

## Sequence

```
     FrontendApp          ViewdefStore                View            ProtocolHandler
        |                      |                      |                      |
        |---watch(1)---------->|                      |                      |
        |                      |                      |                      |
        |<--update(1,{viewdefs:|                      |                      |
        |   {T.NS: html,...}}) |                      |                      |
        |                      |                      |                      |
        |     [for each TYPE.NAMESPACE in viewdefs]   |                      |
        |---validate(T.NS,html)|                      |                      |
        |                      |                      |                      |
        |                      |---parseHtml--------->|                      |
        |                      |   (innerHTML)        |                      |
        |                      |                      |                      |
        |                      |---checkRootElement-->|                      |
        |                      |   (single element?)  |                      |
        |                      |                      |                      |
        |                      |---checkIsTemplate--->|                      |
        |                      |   (is <template>?)   |                      |
        |                      |                      |                      |
        |                      |          [if validation fails]              |
        |                      |---sendError----------|--------------------->|
        |                      |   (varId:1, desc)    |                      |
        |                      |                      |                      |
        |                      |          [if validation passes]             |
        |---store(T.NS,        |                      |                      |
        |    template)-------->|                      |                      |
        |                      |                      |                      |
        |     [process pending views]                 |                      |
        |                      |---processPending---->|                      |
        |                      |                      |                      |
        |                      |     [for each pending view]                 |
        |                      |                      |---render()---------->|
        |                      |                      |                      |
        |                      |          [if returns true]                  |
        |                      |                      |---removePending----->|
        |                      |                      |                      |
        |                      |          [if returns false, stays pending]  |
        |                      |                      |                      |
        |     [subsequent updates]                    |                      |
        |<--update(1,{viewdefs:|                      |                      |
        |   {NEW.NS: html}})   |                      |                      |
        |                      |                      |                      |
        |---validate+store---->|                      |                      |
        |---processPending---->|                      |                      |
        |                      |                      |                      |
```

## Notes

- Viewdef format: single `<template>` element as root
- Validation: parse HTML, verify exactly one root, root is template
- If validation fails, send error message to backend (varId: 1)
- Viewdefs stored by TYPE.NAMESPACE key (e.g., Contact.DEFAULT)
- Previous viewdefs property values can be replaced (frontend stores separately)
- After storing new viewdefs, attempt to render pending views
- Pending views removed from list when render() returns true
- Viewdefs delivered with :high priority to ensure availability before use
