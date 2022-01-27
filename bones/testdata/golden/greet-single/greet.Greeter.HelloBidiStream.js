// greet.Greeter.HelloBidiStream (Bidirectional streaming)

// Input:
// {
//   request: {
//     firstName: '',  // string
//   },
// }

function HelloBidiStream(input) {
  return {
    stream: [
      {
        greeting: '',  // string
      },
    ],
  }
}
