export const getReqType = (t) => {
  switch (t) {
    case "get":
      return <p className="text-green-400 font-bold">GET</p>;
    case "post":
      return <p className="text-yellow-400 font-bold">POS</p>;
    case "delete":
      return <p className="text-red-400 font-bold">DEL</p>;
    case "put":
      return <p className="text-blue-400 font-bold">PUT</p>;
    default:
      return <p className="text-emerald-500 font-bold">GET</p>;
  }
};
