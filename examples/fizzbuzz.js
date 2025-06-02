/**
 * FizzBuzz Implementation
 * 
 * Prints numbers from 1 to 100, but:
 * - For multiples of 3, prints "Fizz" instead of the number
 * - For multiples of 5, prints "Buzz" instead of the number  
 * - For multiples of both 3 and 5, prints "FizzBuzz" instead of the number
 */

function fizzbuzz(limit = 100) {
    for (let i = 1; i <= limit; i++) {
        let output = '';
        
        if (i % 3 === 0) {
            output += 'Fizz';
        }
        
        if (i % 5 === 0) {
            output += 'Buzz';
        }
        
        // If no Fizz or Buzz was added, use the number
        if (output === '') {
            output = i;
        }
        
        console.log(output);
    }
}

// Run the classic FizzBuzz for numbers 1-100
console.log('FizzBuzz (1-100):');
fizzbuzz();

console.log('\n--- Alternative Implementation ---\n');

// Alternative implementation using array mapping
function fizzbuzzArray(limit = 100) {
    return Array.from({ length: limit }, (_, i) => {
        const num = i + 1;
        const fizz = num % 3 === 0 ? 'Fizz' : '';
        const buzz = num % 5 === 0 ? 'Buzz' : '';
        return fizz + buzz || num;
    });
}

// Demonstrate the array version
console.log('FizzBuzz Array (1-20):');
console.log(fizzbuzzArray(20));

// Export for use in other modules (if running in Node.js)
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { fizzbuzz, fizzbuzzArray };
}