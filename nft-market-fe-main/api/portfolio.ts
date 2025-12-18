import { request } from "./request";

type GetPortfolioParams = {
  filters: {
    user_addresses: string[];
  };
};
function GetPortfolio(params: GetPortfolioParams) {
  return request.get("/portfolio/collections", {
    params: {
      filters: JSON.stringify(params.filters),
    },
  });
}


export default {
  GetPortfolio,
};
