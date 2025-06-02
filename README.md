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
      - [] Backspace
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