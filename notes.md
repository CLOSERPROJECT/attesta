# Specific context

We had a meeting with a client who wants to **integrate attesta in their workflow** (Feragame)

## Feragame

Company that extracts materials from solar panels

### Workflow

- They have an extraction "line" that works continuously
- Every **station** along the line extracts a different material
- Each station has materials containers that cyclically are filled, stocked, and replaced with empty ones
- Operators register their weight before stock.
  - **This is the information that needs to be registered in attesta**
- Containers are called, in italian: Unità di Carico (**UdC**)

## Issues

### Single org flow

- Consideration: attesta was born to document multi-org flows
- Need: now we need to track only a single org
- Potential solution: in a single "step", only one org is owner

### Non-linearity of information registration

- **Issue**: right now substeps are thought to work "sequentially".
  - Each substep, once completed, unlocks the next one
  - Reason: they were made to represent the order in which operations happen in the organization
  - Contrast: **UdC** get filled in various rates
    - e.g.: **station 1** fills its UdC, then **station 2**, then station 1 again, then 3
  - Feragame need: operators should be able to update easily all the various steps, then at the end of shift commit the whole step

# General considerations

- Attesta could have different levels of "detail":
  - It makes sense that steps are sequential, because it's one-org-after-the-other
  - But inside one org step, things can happen differently

## Directive

- Investigate current context
- Design 3 solutions:
  - One that changes less code as possible
  - One that introduces a proper architecture design
  - A third solution of your choice. design this afterwards, as a compromise of the first two
