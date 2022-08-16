// httpgreet.HttpGreeter.SimpleHello (Unary)

// Input:
// {
//   request: {
//     firstName: "",  // string
//     lastName: "",  // string
//   },
// }

function(input, metadata) {
  response: {
    greeting: "Simply, hello, " + input.request.firstName,  // string
  },
}
