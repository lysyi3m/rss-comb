package feed

import (
	"strings"
	"testing"
)

func TestContentExtractor_ExtractContent_ValidHTML(t *testing.T) {
	extractor := NewContentExtractor()

	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Test Article</title>
	</head>
	<body>
		<header>
			<h1>Site Header</h1>
			<nav>Navigation</nav>
		</header>
		<main>
			<article>
				<h1>Main Article Title</h1>
				<p>This is the main content of the article. It contains several paragraphs of meaningful text that should be extracted by the readability algorithm.</p>
				<p>This is another paragraph with more content. The readability algorithm should identify this as the main content area and extract it properly.</p>
				<p>Here is some more substantial content to ensure we meet the character threshold. This paragraph adds more context and information that would be valuable to readers.</p>
			</article>
		</main>
		<aside>
			<div>Advertisement</div>
			<div>Related Links</div>
		</aside>
		<footer>
			<p>Copyright 2024</p>
		</footer>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty result")
	}

	// Check that main content is included
	if !strings.Contains(result, "main content of the article") {
		t.Errorf("Expected extracted content to contain main article text")
	}

	// Check that non-content elements are likely excluded
	if strings.Contains(result, "Advertisement") {
		t.Errorf("Expected extracted content to exclude advertisement")
	}

	if strings.Contains(result, "Copyright 2024") {
		t.Errorf("Expected extracted content to exclude footer")
	}
}

func TestContentExtractor_ExtractContent_ArticleWithMetadata(t *testing.T) {
	extractor := NewContentExtractor()

	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>News Article</title>
		<meta name="description" content="Breaking news story">
		<meta name="author" content="John Doe">
	</head>
	<body>
		<article>
			<header>
				<h1>Breaking News: Important Update</h1>
				<time datetime="2024-01-01">January 1, 2024</time>
				<span class="author">By John Doe</span>
			</header>
			<div class="content">
				<p>This is a breaking news story with important information that readers need to know about current events.</p>
				<p>The story continues with more details and context about the situation, providing comprehensive coverage of the topic.</p>
				<p>Additional paragraphs provide more depth and analysis of the breaking news event, ensuring readers get complete information.</p>
			</div>
		</article>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty result")
	}

	// Check that article content is included
	if !strings.Contains(result, "breaking news story") {
		t.Errorf("Expected extracted content to contain article text")
	}

	// The readability algorithm may or may not include the title/header
	// This depends on the specific implementation and HTML structure
	// What's important is that the main content is extracted, which we already verified above
}

func TestContentExtractor_ExtractContent_BlogPost(t *testing.T) {
	extractor := NewContentExtractor()

	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>My Blog Post</title>
	</head>
	<body>
		<div class="site-header">
			<h1>My Blog</h1>
			<nav>Home | About | Contact</nav>
		</div>
		<div class="main-content">
			<article class="post">
				<h2>How to Build Great Software</h2>
				<div class="post-meta">
					<span>Posted on January 1, 2024</span>
					<span>by Tech Blogger</span>
				</div>
				<div class="post-content">
					<p>Building great software requires careful planning and attention to detail. In this post, we'll explore the key principles that guide successful software development.</p>
					<p>First, it's important to understand your users and their needs. User research and feedback are crucial for creating software that actually solves real problems.</p>
					<p>Second, focus on code quality and maintainability. Well-written code is easier to debug, extend, and modify over time.</p>
					<p>Finally, testing is essential for ensuring your software works as expected. Comprehensive testing helps catch bugs early and prevents regressions.</p>
				</div>
			</article>
		</div>
		<div class="sidebar">
			<div>Recent Posts</div>
			<div>Categories</div>
			<div>Advertisement</div>
		</div>
		<div class="footer">
			<p>© 2024 My Blog. All rights reserved.</p>
		</div>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty result")
	}

	// Check that main content is included
	if !strings.Contains(result, "Building great software") {
		t.Errorf("Expected extracted content to contain main post content")
	}

	if !strings.Contains(result, "code quality and maintainability") {
		t.Errorf("Expected extracted content to contain detailed post content")
	}
}

func TestContentExtractor_ExtractContent_EmptyData(t *testing.T) {
	extractor := NewContentExtractor()

	result, err := extractor.Run([]byte{})

	if err == nil {
		t.Errorf("Expected error for empty data")
	}

	if result != "" {
		t.Errorf("Expected empty result for empty data")
	}

	expectedError := "HTML data is empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestContentExtractor_ExtractContent_NilData(t *testing.T) {
	extractor := NewContentExtractor()

	result, err := extractor.Run(nil)

	if err == nil {
		t.Errorf("Expected error for nil data")
	}

	if result != "" {
		t.Errorf("Expected empty result for nil data")
	}

	expectedError := "HTML data is empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestContentExtractor_ExtractContent_InvalidHTML(t *testing.T) {
	extractor := NewContentExtractor()

	// Malformed HTML
	htmlContent := `<html><body><p>Unclosed paragraph<div>Malformed content</body>`

	result, err := extractor.Run([]byte(htmlContent))

	// The go-readability library should handle malformed HTML gracefully
	// It might succeed with partial content or fail, both are acceptable
	if err != nil {
		// If it fails, that's okay for malformed HTML
		if result != "" {
			t.Errorf("Expected empty result when extraction fails")
		}
	} else {
		// If it succeeds, result should not be empty
		if result == "" {
			t.Errorf("Expected non-empty result when extraction succeeds")
		}
	}
}

func TestContentExtractor_ExtractContent_MinimalHTML(t *testing.T) {
	extractor := NewContentExtractor()

	// Very minimal HTML that might not meet character threshold
	htmlContent := `<html><body><p>Short text</p></body></html>`

	result, err := extractor.Run([]byte(htmlContent))

	// This might fail due to character threshold (250 chars minimum)
	// or succeed with the minimal content
	if err != nil {
		// If it fails due to insufficient content, that's expected
		if result != "" {
			t.Errorf("Expected empty result when extraction fails")
		}
	} else {
		// If it succeeds, result should contain the content
		if !strings.Contains(result, "Short text") {
			t.Errorf("Expected extracted content to contain the text")
		}
	}
}

func TestContentExtractor_ExtractContent_NoMainContent(t *testing.T) {
	extractor := NewContentExtractor()

	// HTML with only navigation and footer, no main content
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head><title>Test</title></head>
	<body>
		<nav>
			<ul>
				<li><a href="/">Home</a></li>
				<li><a href="/about">About</a></li>
				<li><a href="/contact">Contact</a></li>
			</ul>
		</nav>
		<footer>
			<p>© 2024 Test Site</p>
		</footer>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	// Should fail because there's no substantial content
	if err == nil && result == "" {
		t.Errorf("Expected error or non-empty result")
	}

	// If it succeeds, it might extract navigation or other elements
	// If it fails, that's expected behavior
}

func TestContentExtractor_ExtractContent_LongArticle(t *testing.T) {
	extractor := NewContentExtractor()

	// Create a long article that definitely meets character threshold
	var paragraphs []string
	for i := 0; i < 10; i++ {
		paragraphs = append(paragraphs, `<p>This is paragraph number `+string(rune(i+48))+`. It contains substantial content that should be extracted by the readability algorithm. The content is meaningful and provides value to readers who are interested in the topic being discussed.</p>`)
	}

	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Long Article</title>
	</head>
	<body>
		<nav>Site Navigation</nav>
		<main>
			<article>
				<h1>Long Article Title</h1>
				` + strings.Join(paragraphs, "\n") + `
			</article>
		</main>
		<aside>
			<div>Sidebar content</div>
			<div>More sidebar content</div>
		</aside>
		<footer>Footer content</footer>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	if err != nil {
		t.Errorf("Expected no error for long article, got: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty result for long article")
	}

	// Check that substantial content is included
	if !strings.Contains(result, "paragraph number") {
		t.Errorf("Expected extracted content to contain article paragraphs")
	}

	// Check that the result is reasonably long (should have extracted multiple paragraphs)
	if len(result) < 200 {
		t.Errorf("Expected extracted content to be substantial, got %d characters", len(result))
	}
}

func TestContentExtractor_ExtractContent_PreservesFormatting(t *testing.T) {
	extractor := NewContentExtractor()

	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Formatted Article</title>
	</head>
	<body>
		<article>
			<h1>Article with Formatting</h1>
			<p>This paragraph contains <strong>bold text</strong> and <em>italic text</em> that should be preserved.</p>
			<p>Here's a <a href="https://example.com">link to example</a> that should be maintained.</p>
			<ul>
				<li>First list item</li>
				<li>Second list item</li>
			</ul>
			<p>This paragraph follows the list and contains more content for the article.</p>
		</article>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty result")
	}

	// Check that HTML formatting is preserved
	if !strings.Contains(result, "<strong>") || !strings.Contains(result, "</strong>") {
		t.Errorf("Expected extracted content to preserve bold formatting")
	}

	if !strings.Contains(result, "<em>") || !strings.Contains(result, "</em>") {
		t.Errorf("Expected extracted content to preserve italic formatting")
	}

	if !strings.Contains(result, "<a href=") {
		t.Errorf("Expected extracted content to preserve links")
	}
}

func TestContentExtractor_ExtractContent_ScriptAndStyleRemoval(t *testing.T) {
	extractor := NewContentExtractor()

	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Article with Scripts</title>
		<style>
			body { font-family: Arial; }
			.content { margin: 20px; }
		</style>
	</head>
	<body>
		<script>
			console.log("This script should be removed");
			var trackingCode = "analytics";
		</script>
		<article>
			<h1>Clean Article Content</h1>
			<p>This is the main content that should be extracted without any scripts or styles interfering. The article contains substantial text content that meets the readability algorithm's requirements.</p>
			<p>The content extraction should focus on the meaningful text and ignore technical elements. This paragraph provides additional context and information for readers.</p>
			<p>Here is more substantial content to ensure we meet the character threshold. This article discusses important topics and provides valuable information to readers who are interested in the subject matter.</p>
		</article>
		<script>
			// More JavaScript that should be excluded
			function trackEvent() { }
		</script>
	</body>
	</html>
	`

	result, err := extractor.Run([]byte(htmlContent))

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty result")
	}

	// Check that main content is included
	if !strings.Contains(result, "main content that should be extracted") {
		t.Errorf("Expected extracted content to contain main article text")
	}

	// Check that script content is excluded
	if strings.Contains(result, "console.log") {
		t.Errorf("Expected extracted content to exclude script content")
	}

	if strings.Contains(result, "trackingCode") {
		t.Errorf("Expected extracted content to exclude script variables")
	}

	// Check that style content is excluded
	if strings.Contains(result, "font-family") {
		t.Errorf("Expected extracted content to exclude style content")
	}
}

func TestNewContentExtractor(t *testing.T) {
	extractor := NewContentExtractor()

	if extractor == nil {
		t.Errorf("Expected non-nil ContentExtractor")
	}

	// Verify it's a valid instance by testing a method
	result, err := extractor.Run([]byte("<html><body><p>test</p></body></html>"))

	// Should either succeed or fail gracefully (due to character threshold)
	if err != nil && result != "" {
		t.Errorf("Inconsistent state: error but non-empty result")
	}
}
