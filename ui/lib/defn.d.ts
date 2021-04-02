declare module "qr.js" {
  enum ErrorCorrectLevel {
    L = 1,
    M = 0,
    Q = 3,
    H = 2,
  }
  type QROptions = {
    errorCorrectLevel?: ErrorCorrectLevel;
  };
  type gen = {
    (data: string, options?: QROptions): {
      modules: boolean[][];
    };
    ErrorCorrectLevel: typeof ErrorCorrectLevel;
  };
  const qrcode: gen;
  export = qrcode;
}
