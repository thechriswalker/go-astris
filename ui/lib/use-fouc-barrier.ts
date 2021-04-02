import { useEffect, useState } from "react";

export function useFoucBarrier(ms: number = 100): true | null {
  const [ok, setOK] = useState<true | null>(null);
  useEffect(() => {
    const t = setTimeout(() => {
      setOK(true);
    }, ms);
    return () => {
      clearTimeout(t);
    };
  }, []);
  return ok;
}
