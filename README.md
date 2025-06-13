# Gowser
A Toy Web Browser based on [Web Browser Engineering](https://browser.engineering/)

## Progress

1. Downloading Web Pages
   - [X] URL Parsing
   - [X] Connecting to Host
     - [x] Encryption
   - [X] Send HTTP Request
   - [X] Receive and Split HTTP Response
   - [X] Print Text
   - [ ] Exercises (Optional)
     - [ ] HTTP/1.1
     - [ ] File URLs
     - [ ] data
     - [ ] Entities
     - [ ] view-source
     - [ ] Keep-alive
     - [ ] Redirects
     - [ ] Caching
     - [ ] Compression

2. Drawing to the Screen
   - [x] Window Creating
   - [x] Text Layout and Drawing
   - [x] Listening to Key Events
   - [x] Scrolling the content in the window
   - [ ] Exercises (Optional)
     - [ ] Line breaks
     - [ ] Mouse wheel
     - [ ] Resizing
     - [ ] Scrollbar
     - [ ] Zoom
     - [ ] Emoji
     - [ ] about:blank
     - [ ] Alternate text direction

3. Formatting Text
    - [x] Text Layout word by word
    - [x] Split lines at words
    - [x] Text can be bold and italic
    - [x] Text in different sizes can be mixed
    - [X] Font Cache
    - [ ] Exercises (Optional)
      - [ ] Centered Text
      - [ ] Superscripts
      - [ ] Soft hyphens
      - [ ] Small caps
      - [ ] Preformatted text

4. Constructing an HTML Tree
    - [x] HTML parser
    - [x] Handling attributes
    - [x] Some fixes for malformed HTML
    - [x] Recursive layout algorithm for tree
    - [ ] Exercises (Optional)
      - [ ] Comments
      - [ ] Paragraphs
      - [ ] Scripts
      - [ ] Quoted attributes
      - [ ] Syntax highlighting
      - [ ] Mis-nested formatting tags

5. Laying Out Pages
    - [x] Tree based layout
    - [x] Layoutmodes in nodes (block/inline)
    - [x] Layout computes size and position
    - [x] Displaylist contains commands
    - [x] Source code snippets have background
    - [ ] Exercises (Optional)
      - [ ] Links bar
      - [ ] Hidden head
      - [ ] Bullets
      - [ ] Table of Contents
      - [ ] Anonymous block boxes
      - [ ] Run-ins

6. Applying Author Styles
    - [x] Add CSS parser
    - [x] Add support for style attributes and linked CSS files
    - [x] Implement cascading and inheritance
    - [x] Refactor BlockLayout to move the font properties to CSS
    - [x] Move most tag-specific reasoning to a browser style sheet
    - [ ] Exercises (Optional)
      - [ ] Fonts (font-family)
      - [ ] Width/Height
      - [ ] Class selectors
      - [ ] display property
      - [ ] Shorthand Properties
      - [ ] Inline Style Sheets
      - [ ] Fast Descendant Selectors
      - [ ] Selector Sequences
      - [ ] !important
      - [ ] :has selectors

7. Handling Buttons and Links
    - [x] Size and Position for each word
    - [x] Determine where the user clicks
    - [x] Split up browser with tabs
    - [x] Draw tabs, address bar and more
    - [x] Implement text editing
    - [ ] Exercises (Optional)
      - [x] Backspace
      - [ ] Middle click
      - [ ] Window title
      - [ ] Forward
      - [ ] Fragments
      - [ ] Search
      - [ ] Visited links
      - [ ] Bookmarks
      - [ ] Cursor
      - [ ] Multiple windows
      - [ ] Clicks via the display list

8. Sending Information to Servers
    - [x] Layout for input and buttons
    - [x] Click on buttons and type into inputs
    - [x] Hierarchical focus handling
    - [x] Submit forms to server
    - [x] Small server to handle forms
    - [ ] Exercises (Optional)
      - [ ] Enter key
      - [ ] GET forms
      - [ ] Blurring
      - [ ] Check boxes
      - [ ] Resubmit requests
      - [ ] Message board
      - [ ] Persistence
      - [ ] Rich buttons
      - [ ] HTML chrome

9.  Running Interactive Scripts
    - [x] Basic javascript support
    - [x] Generating handles for scripts to refer to page elements
    - [x] Reading attribute values from page elements
    - [x] Writing and modifying page elements
    - [x] Attaching event listeners for scripts to respond to page events
    - [ ] Exercises (optional)
      - [ ] Node.children
      - [ ] createElement / appendChild / insertBefore
      - [ ] removeChild
      - [ ] IDs
      - [ ] Event bubbling
      - [ ] Serializing HTML
      - [ ] Script-added scripts and style sheets

10. Keeping Data Private
    - [x] Mitigating cross-site XMLHttpRequests with the same-origin policy
    - [x] Mitigating cross-site request forgery with nonces and with SameSite cookies
    - [x] mitigating cross-site scripting with escaping and with Content-Security-Policy
    - [ ] Exercises (optional)
      - [ ] New inputs (hidden, password)
      - [ ] Certificate errors
      - [ ] Script access
      - [ ] Cookie expiration
      - [ ] Cross-origin resource sharing
      - [ ] Referer

11. Adding Visual Effects
    - [x] Browser compositing with extra surfaces for faster scrolling
    - [x] Partial transparency via an alpha channel
    - [x] User-configurable blending modes via mix-blend-mode
    - [ ] Rounded rectangle clipping via destination-in blending or direct clipping
    - [x] Optimizations to avoid surfaces when possible
    - [ ] Exercises (optional)
      - [ ] Filters
      - [ ] Hit testing
      - [ ] Interest Region
      - [ ] Overflow scrolling
      - [ ] Touch input

12. Scheduling Tasks and Threads
    - [x] Task queues with tasks for js, user input and rendering
    - [x] Trying to consistently generate frames at fixed interval, i.e. 30hz
    - [x] Two threads involved in rendering
    - [x] Main thread runs javascript and special rendering
    - [x] Browser thread draws display list to screen, handles input events and scrolling
    - [x] Main thread synchronizes with browser thread through commit
    - [ ] Exercises (optional)
      - [ ] setInterval
      - [ ] Task timing
      - [ ] Clock-based frame timing
      - [ ] Scheduling
      - [ ] Threaded loading
      - [ ] Networking thread
      - [ ] Optimized scheduling
      - [ ] Raster-and-draw thread

13. Animating and Compositing
    - [x] Different types of Animations (DOM, input-driven, etc)
    - [ ] GPU Acceleration
    - [x] Compositing for smooth and threaded visual effect animations
    - [x] Optimized compositing layers
    - [x] Transformed elements
      - [x] Overlap testing
    - [ ] Exercises (optional)
      - [ ] background-color
      - [ ] Easing functions
      - [ ] Composited and threaded animations
      - [ ] Width/Height animations
      - [ ] CSS animations
      - [ ] Overlap testing with transform animations
      - [ ] Avoiding sparse composited layers
      - [ ] Short display lists
      - [ ] Hit testing
      - [ ] z-index
      - [ ] Animated scrolling
      - [ ] Opacity plus draw

14. Making Content Accessible
15. Supporting Embedded Content
16. Reusing Previous Computation