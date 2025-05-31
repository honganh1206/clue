// ROT13 decoder function
function rot13(str) {
    return str.replace(/[a-zA-Z]/g, function(char) {
        const start = char <= 'Z' ? 65 : 97; // ASCII value for 'A' or 'a'
        return String.fromCharCode(((char.charCodeAt(0) - start + 13) % 26) + start);
    });
}

// The ROT13 encoded string
const encodedString = "Pbatenghyngvbaf ba ohvyqvat n pbqr-rqvgvat ntrag!";

// Decode and print the message
const decodedMessage = rot13(encodedString);
console.log(decodedMessage);