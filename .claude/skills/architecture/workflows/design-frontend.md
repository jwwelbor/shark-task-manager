# Frontend Architecture Design Workflow

This workflow guides you through creating comprehensive frontend architecture documentation for a feature. It produces the frontend design document (05-frontend-design.md) with component architecture, state management, and UX patterns.

## Prerequisites

Before starting this workflow, ensure you have:
1. Feature PRD at `/docs/plan/{epic-key}/{feature-key}/prd.md`
2. Research report at `/docs/plan/{epic-key}/{feature-key}/00-research-report.md`
3. Interface contracts (if defined by feature-architect)
4. Backend design (04-backend-design.md) for API integration details

## Step 1: Determine UI Design Approach

### Assess if Frontend-Design Skill is Needed

**Invoke frontend-design skill when:**
- Feature requires user-facing web pages or applications
- Interactive UI components need visual polish
- Building dashboards, forms, or data visualization
- PRD emphasizes user experience or visual design quality

**Skip frontend-design skill when:**
- Feature is API-only or backend-focused
- Components are purely structural (no visual design)
- Reusing existing components without changes

### If Using Frontend-Design Skill
1. Invoke the skill first to get design guidance
2. Incorporate visual design decisions into architecture doc
3. Reference the skill's component specifications
4. Use the skill's layout and styling patterns

## Step 2: Analyze Requirements

### Read the PRD
- Identify user stories and user flows
- Extract UI requirements
- Note interactivity requirements
- Identify data display needs
- Understand accessibility requirements

### Read the Research Report
- Understand frontend framework in use (React, Vue, Angular, Svelte)
- Note component organization patterns
- Identify state management approach (Redux, Vuex, Context, Pinia)
- Review naming conventions
- Find existing similar components to extend

### Review Backend Design
- Understand available API endpoints
- Review DTOs for request/response structures
- Note authentication/authorization requirements
- Identify data fetching patterns

## Step 3: Design Component Architecture

### Define Component Hierarchy

Create a Mermaid diagram showing:
- **Pages** - Top-level route components
- **Organisms** - Complex composite components
- **Molecules** - Reusable component groups
- **Atoms** - Basic UI elements

Follow Atomic Design principles:
- Pages compose organisms
- Organisms compose molecules
- Molecules compose atoms
- Clear data flow: props down, events up

### Define Component Data Flow

Create a Mermaid diagram showing:
- Store/state management
- Page components
- Child components
- Data flow direction (state → props, events → actions)

## Step 4: Specify Each Component

For each component (Page, Organism, Molecule, Atom):

### Component Specification Structure

#### Basic Info
- **Type**: Page / Organism / Molecule / Atom
- **Purpose**: What it does and when it's used
- **Route** (Pages only): URL path and parameters

#### Props
- `propName`: type description - purpose
- Mark optional props explicitly
- Note default values if applicable

#### Local State
- `stateName`: type description - what triggers changes
- Keep state minimal (lift state when shared)

#### Events Emitted
- `eventName`: When emitted, payload structure
- Follow project event naming conventions

#### Child Components
- List child components and how they're used
- Note data passed to children

#### Behavior
- On mount actions
- User interaction responses
- State change reactions
- API calls triggered

#### Accessibility
- ARIA role
- Keyboard interactions
- Screen reader support
- Focus management

## Step 5: Design State Management

### Store Architecture

Create a Mermaid diagram showing:
- State stores for this feature
- Actions that modify state
- Getters/computed values
- API layer integration
- Component subscriptions

### State Shape

Define state structure:
- Feature-specific properties
- Loading states (boolean)
- Error states (string or null)
- Derived/computed state (via getters)

### Getters (Computed Properties)

Define computed values:
- What they compute
- What state they derive from
- When to use them

### Actions

Define state-modifying actions:
- Action name and parameters
- What API it calls
- What state it updates
- Error handling

### Store Dependencies

Document:
- Other stores this depends on
- Components that use this store
- Shared state concerns

## Step 6: Design API Integration

### Endpoints Used

Create a table:
- Endpoint path
- HTTP method
- Request DTO
- Response DTO
- Which component uses it

### Data Flow Diagrams

For primary user actions, create Mermaid sequence diagrams:
- User interaction
- Component action
- Store dispatch
- API call
- Response handling
- State update
- UI reactive update

### Error Handling

Document how to handle:
- Network errors
- Validation errors (422)
- Authentication errors (401/403)
- Server errors (500)
- Error display patterns (toast, inline, modal)

## Step 7: Define UX Patterns

### Loading States
- Initial load (skeleton, spinner, placeholder)
- Action pending (inline indicators)
- Background refresh (non-blocking indicators)

### Form Handling
- Client-side validation approach
- Submission (optimistic vs. wait for response)
- Error display (inline, toast, summary)
- Success feedback (toast, redirect, inline)

### Empty States
- No data available
- No search/filter results
- Error state with recovery options

### Optimistic Updates (if applicable)
- Which actions use optimistic UI
- Rollback strategy on failure

## Step 8: Design Responsive Behavior

### Breakpoints

Define for:
- Mobile (< 640px)
- Tablet (640-1024px)
- Desktop (> 1024px)

### Component Adaptations

For each major component:
- How layout changes across breakpoints
- Mobile-specific interactions
- Touch vs. mouse considerations

## Step 9: Ensure Accessibility (WCAG 2.1 AA)

### Keyboard Navigation
- Logical tab order
- Focus management on actions
- Keyboard shortcuts (if any)

### Screen Reader Support
- ARIA labels for non-semantic elements
- Live regions for dynamic content
- Semantic HTML usage

### Visual Accessibility
- Color contrast requirements
- Don't rely on color alone
- Visible focus indicators
- Text sizing and scalability

## Step 10: Define Testing Strategy

### Component Tests
- Key behaviors to test per component
- Edge cases and error states
- Prop variations

### Integration Tests
- User flows to test end-to-end
- API integration verification

### Visual Regression
- Key components/pages to capture

## Step 11: Document Visual Design

If frontend-design skill was used:
- Include key design decisions
- Note design system alignment
- Document new patterns introduced
- Specify color and typography approach
- Define visual hierarchy

If NOT using frontend-design skill:
- Document alignment with existing design system
- Note any design patterns being followed

## Step 12: Quality Checklist

Before finalizing, verify:

### Completeness
- [ ] All required template sections filled
- [ ] Document is 150-200 lines
- [ ] All components fully specified
- [ ] State management completely defined
- [ ] All API integrations documented
- [ ] UX patterns for all states defined

### NO CODE Constraint
- [ ] No Vue/React/Angular code
- [ ] No TypeScript interfaces
- [ ] No CSS/SCSS styles
- [ ] No JavaScript functions
- [ ] Only prose descriptions and specs
- [ ] Mermaid diagrams for visualization

### Consistency
- [ ] Follows patterns from research report
- [ ] Component naming matches conventions
- [ ] Props align with backend DTOs
- [ ] State management follows project pattern

### Accessibility
- [ ] WCAG 2.1 AA compliance addressed
- [ ] Keyboard navigation defined
- [ ] Screen reader support specified
- [ ] Visual accessibility covered

### Integration
- [ ] Props match backend response DTOs
- [ ] API endpoints align with backend design
- [ ] Error handling covers all cases
- [ ] Cross-references to backend docs

## Step 13: Create Document

### File Location
Create `/docs/plan/{epic-key}/{feature-key}/05-frontend-design.md`

### Use Template
Follow `context/templates/frontend-doc.md` structure exactly

### Review
- Verify file is complete
- Check all cross-references are valid
- Ensure diagrams render correctly
- Confirm no code blocks exist
- Validate accessibility coverage

## Best Practices to Apply

### Component Design
- **Single Responsibility**: Each component does one thing well
- **Props Down, Events Up**: Clear unidirectional data flow
- **Composition Over Inheritance**: Build complex UIs from simple pieces
- **Controlled Components**: Be explicit about form control patterns

### State Management
- **Minimize Global State**: Only lift state when necessary
- **Normalize Data**: Avoid deeply nested state
- **Derive Don't Duplicate**: Use getters for computed values
- **Immutable Updates**: Never mutate state directly

### Performance
- **Lazy Loading**: Routes and heavy components
- **Memoization**: Expensive computations
- **Virtualization**: Long lists
- **Optimize Re-renders**: Prevent unnecessary updates

### Accessibility
- **Semantic HTML First**: Use proper elements
- **ARIA When Needed**: Only when semantic HTML isn't enough
- **Test with Screen Readers**: Verify actual experience
- **Keyboard-Only Navigation**: Support all interactions

## Output Requirements

Upon completion, you will have:
1. **05-frontend-design.md** - Complete frontend architecture
2. Component hierarchy with full specifications
3. State management architecture
4. API integration patterns
5. UX patterns for all states
6. Responsive design specifications
7. Accessibility requirements
8. Testing strategy
9. Document following template exactly
10. NO implementation code, only design specifications
