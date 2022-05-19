// httpgreet.HttpGreeter.SimpleHello (Unary)

// Input:
// {
//   request: {
//     firstName: "",  // string
//     lastName: "",  // string
//   },
// }

function(input) {
  response: {
    greeting: "Simply, hello, " + input.request.firstName,  // string
  },
}
