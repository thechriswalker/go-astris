import * as React from "react";

export const Section: React.FC<{
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
}> = ({ title, subtitle, children }) => {
  return (
    <section className="section">
      {title && <h1 className="title">{title}</h1>}
      {subtitle && <h2 className="subtitle">{subtitle}</h2>}
      {children}
    </section>
  );
};
