import React from "react";
import { DateTime } from "luxon";
import cx from "classnames";

import { TimeSpec, TimeWindow, TimingData } from "../lib/types";
import { SetupSection } from "./types";
import { Section } from "../components/Section";

export const SetupTiming: SetupSection = ({ setup, update }) => {
  setup.timing ??= {
    timeZone: localTimezone(),
  } as TimingData;

  const timeupdate = (key: keyof Omit<TimingData, "timeZone">) => (
    timeWindow: TimeWindow
  ) => {
    setup.timing[key] = timeWindow;
    update({ ...setup });
  };

  return (
    <Section
      title="Step 2: Election Timing"
      subtitle="Configure the timing of the stages of the election."
    >
      <div className="container">
        <div className="box">
          <div className="field is-horizontal">
            <div className="field-label is-medium">
              <label className="label">Election Timezone</label>
            </div>
            <div className="field-body">
              <div className="field is-expanded">
                <div className="control">
                  <input
                    className="input is-medium"
                    type="text"
                    value={setup.timing.timeZone}
                    onChange={(evt) => {
                      setup.timing.timeZone = evt.target.value;
                      update({ ...setup });
                    }}
                  />
                </div>
                <p className="help">
                  This is the timezone that all the following date/times will be
                  resolved in.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className="container mt-4">
        <div className="columns">
          <div className="column">
            <TimeSpecInput
              name="Phase #1: Trustee Confirmation"
              win={setup.timing?.parameterConfirmation}
              onChange={timeupdate("parameterConfirmation")}
            />
            <TimeSpecInput
              win={setup.timing?.voterRegistration}
              prev={setup.timing?.parameterConfirmation?.closes}
              name="Phase #2: Voter Registration"
              onChange={timeupdate("voterRegistration")}
            />
          </div>
          <div className="column">
            <TimeSpecInput
              win={setup.timing?.voteCasting}
              prev={setup.timing?.voterRegistration?.closes}
              name="Phase #3: Vote Casting"
              onChange={timeupdate("voteCasting")}
            />
            <TimeSpecInput
              win={setup.timing?.tallyDecryption}
              prev={setup.timing?.voteCasting?.closes}
              name="Phase #4: Trustee Tallying"
              onChange={timeupdate("tallyDecryption")}
            />
          </div>
        </div>
      </div>
    </Section>
  );
};

// input for a TimeSpec
// with both Date and Time
const TimeSpecInput: React.FC<{
  name: string;
  win?: TimeWindow;
  prev?: TimeSpec; // <- to identify errors!
  onChange?: (tw: TimeWindow) => unknown;
}> = ({ name, win, prev = localNow(), onChange = () => {} }) => {
  let { opens = "", closes = "" } = win || {};

  // if the values are not empty, they should "parse" as local date times
  // and they should be "later" than the "prev if exists."
  // check the "open" first
  let opensEarly = false;
  let closesEarly = false;

  const prevDt = prev && DateTime.fromISO(prev);
  const openDt = opens && DateTime.fromISO(opens);
  const closeDt = closes && DateTime.fromISO(closes);

  if (prevDt && openDt) {
    // opens is strictly before the prev
    opensEarly = openDt < prevDt;
  }
  if ((openDt || prevDt) && closeDt) {
    // closes is before opens (or prev)
    closesEarly = closeDt <= (openDt || prevDt!);
  }

  return (
    <div className="box">
      <p className="title is-5">{name}</p>
      <div className="field is-horizontal">
        <div className="field-label is-normal">
          <label className="label">Opens</label>
        </div>
        <div className="field-body">
          <div className="field">
            <div className="control">
              <input
                className={cx("input", { "is-danger": opensEarly })}
                type="datetime-local"
                defaultValue={opens}
                onChange={(evt) => {
                  //datetime-local doesn't have seconds.
                  opens = evt.target.value + ":00";
                  if (!closes) {
                    closes = add1day(opens);
                  }
                  onChange({ opens, closes });
                }}
              />
              <p className="help is-danger">
                {opensEarly
                  ? `This date must be later than ${prevDt!.toLocaleString(
                      DateTime.DATETIME_MED
                    )}`
                  : "\u00a0"}
              </p>
            </div>
          </div>
        </div>
      </div>
      <div className="field is-horizontal">
        <div className="field-label is-normal">
          <label className="label">Closes</label>
        </div>
        <div className="field-body">
          <div className="field">
            <div className="control">
              <input
                className={cx("input", { "is-danger": closesEarly })}
                type="datetime-local"
                defaultValue={closes}
                onChange={(evt) => {
                  //datetime-local doesn't have seconds.
                  closes = evt.target.value + ":00";
                  onChange({ opens, closes });
                }}
              />
              <p className="help is-danger">
                {closesEarly
                  ? `This date must be earlier later than ${(
                      openDt || prevDt!
                    ).toLocaleString(DateTime.DATETIME_MED)}`
                  : "\u00a0"}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

const toISOTruncated = {
  suppressMilliseconds: true,
  includeOffset: false,
} as const;

function localTimezone(): string {
  return DateTime.local().zoneName;
}

function localNow(): TimeSpec {
  return DateTime.local().toISO(toISOTruncated);
}

function add1day(ts: TimeSpec): TimeSpec {
  return DateTime.fromISO(ts).plus({ days: 1 }).toISO(toISOTruncated);
}
