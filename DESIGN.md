# DESIGN DIRECTIVES & UI SPECIFICATIONS

## 1. General Principles
This document outlines the visual design, layout, and styling specifications derived from the reference image (`ui-self-docs.jpg`). The goal is to recreate this clean, professional, and accessible interface.

*   **Design Style:** Clean, minimalist, modern flat design with subtle use of shadows for depth.
*   **Color Palette (Inferred from image):**
    *   **Background (Body):** A very soft, light gray or off-white (e.g., `#f8f9fa` or similar). The design uses a full-width light background.
    *   **Cards/Containers:** White (`#ffffff`) to create contrast against the background.
    *   **Text (Headings & Primary):** Dark gray/near black (e.g., `#212529` or `#333333`).
    *   **Text (Secondary/Subtitles):** Mid-gray for descriptions, timestamps, and footer text (e.g., `#6c757d`).
    *   **Accent Color (Buttons):** A vibrant, light lime-green/chartreuse (e.g., `#c4e349` or similar). This is the defining brand color used for call-to-action buttons.
    *   **Accent Color (Icons):** Some icons use a muted golden/yellow tone, while others are standard dark outlines.
    *   **Tags Background:** A very soft gray/blue tint (e.g., `#e9ecef`).
    *   **Borders:** Subtle, light gray lines for dividers (e.g., `#dee2e6`).

*   **Typography:** A modern, clean sans-serif font is used throughout (e.g., Inter, Roboto, or system UI fonts).
    *   Headings are bold and legible.
    *   Body text is highly readable with adequate line height.

## 2. Layout Structure & Components

### 2.1. Header Navigation
*   **Layout:** Full width, with a thin bottom border separating it from the main content. Flexbox row, space-between alignment.
*   **Logo (Left):** An icon (looks like a document/list) next to the text "self-docs" in lowercase, bold font.
*   **Main Navigation (Center):** Horizontal list of links: "Home", "Documents", "Knowledge Base", "Private Notes", "Settings".
    *   "Home" appears to be the active state (perhaps slightly bolder or darker).
*   **User Area (Right):** Notification bell icon (discarded as per PRD) and a user avatar (small circular image) next to the name "A.K." with a dropdown caret icon.

### 2.2. Hero / Welcome Section
*   **Layout:** Centered content, significant vertical padding.
*   **Typography:**
    *   Greeting ("Welcome back, Alex."): `h2` or `h3` size, bold.
    *   Main Title ("Your Personal Knowledge Base."): `h1` size, boldest and largest text on the page.
    *   Subtitle ("A secure and organized hub..."): Smaller, secondary text color.
*   **Search Bar:**
    *   Centered, wide pill shape (fully rounded corners).
    *   Placeholder text: "Search your documentation...".
    *   Contains a circular green button inside the right edge of the input field with a magnifying glass icon and the word "Search".

### 2.3. Main Navigation Cards (Grid)
*   **Layout:** A CSS Grid or Flexbox layout, horizontally aligned. 4 columns on large screens.
*   **Card Styling:**
    *   White background.
    *   Rounded corners (approx. `8px` or `12px`).
    *   Subtle drop shadow (`box-shadow`) to lift them from the background.
    *   Internal padding (approx. `20px` - `24px`).
*   **Card Content (Top to Bottom):**
    *   **Icon:** Contained within a light, rounded-square background. (Book/Folder/List/Padlock icons).
    *   **Title:** Bold, medium size (e.g., `h3` or `h4`).
    *   **Description:** Smaller, secondary text color, two lines of text.
    *   **Button:** Full width of the card's internal content.
        *   Background color: The vibrant lime-green accent.
        *   Text: Black/Dark gray, centered, bold (e.g., "Explore Guides").
        *   Shape: Fully rounded (pill shape).

### 2.4. Lower Content Area (Two Columns)
The layout splits into a roughly 60/40 or 70/30 two-column structure.

**Left Column (Wider): Recently Updated**
*   **Header:** "Recently Updated", bold, left-aligned.
*   **List Items:** Each item is a horizontal row.
*   **Row Content (Left to Right):**
    *   Document Icon.
    *   Document Title (Dark text).
    *   Timestamp (e.g., "2h ago", "Yesterday") - Secondary text color, aligned to the right.
    *   "View" link/button - Small, bold text aligned to the far right.
*   **Dividers:** There are subtle, light gray horizontal lines separating each list item.

**Right Column (Narrower): Popular Tags & Activity Feed**
*   **Popular Tags Section:**
    *   Header: "Popular Tags", bold.
    *   Container: Flexbox with `flex-wrap: wrap` and a small `gap` (e.g., `8px`).
    *   Tag Styling: "Pill" shape (rounded corners), light gray background, dark text (e.g., `#Python`, `#UI_Design`).
*   **Activity Feed Section (Below Tags):**
    *   Header: "Activity Feed", bold.
    *   List Items: Vertical list.
    *   Row Content: Small user avatar next to the activity description text.
    *   Text Formatting: The user's name ("Alex") and the action are standard, while the document title is sometimes bolded or distinct.

### 2.5. Footer
*   **Layout:** Full width, split into columns (looks like 3 main columns).
*   **Column 1 (Left):** "Footer" header (likely a placeholder, perhaps should just be the logo/brand again), copyright info ("© 2024 self-docs"), and inline links ("Privacy Policy Terms Contact").
*   **Column 2 (Center):** "Portal links" header (bold), followed by a vertical list of links in the lime-green accent color ("Admin Dashboard", "Portal API").
*   **Column 3 (Right):** "Useful links" header (bold), followed by the same lime-green links.

## 3. Responsive Considerations (Directives for Development)
While the image shows a desktop layout, development must include responsive states:
*   **Tablet/Mobile:** The 4-column card layout should break down to 2 columns on tablet, and 1 column on mobile.
*   **Two-Column Area:** The "Recently Updated" and the sidebar ("Tags" / "Feed") should stack vertically on smaller screens (sidebar moving below the recent list).
*   **Header:** The navigation links should collapse into a hamburger menu on mobile devices.
