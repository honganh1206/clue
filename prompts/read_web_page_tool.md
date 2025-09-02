Read and analyze the contents of a web page from a given URL.

When only the url parameter is set, it returns the contents of the webpage converted to Markdown.

If the raw parameter is set, it returns the raw HTML of the webpage.

If a prompt is provided, the contents of the webpage and the prompt are passed along to a model to extract or summarize the desired information from the page.

Prefer using the prompt parameter over the raw parameter.

## When to use this tool

- When you need to extract information from a web page (use the prompt parameter)
- When the user shares URLs to documentation, specifications, or reference materials
- When the user asks you to build something similar to what's at a URL
- When the user provides links to schemas, APIs, or other technical documentation
- When you need to fetch and read text content from a website (pass only the URL)
- When you need raw HTML content (use the raw flag)

## When NOT to use this tool

- When visual elements of the website are important - use browser tools instead
- When navigation (clicking, scrolling) is required to access the content
- When you need to interact with the webpage or test functionality
- When you need to capture screenshots of the website

## Examples

<example>
// Summarize key features from a product page
{
  url: "https://example.com/product",
  prompt: "Summarize the key features of this product."
}
</example>
