import * as React from "react";
import { useEffect, useState } from "react";
import { DateTime, Info } from "luxon";
import cx from "classnames";

import { CalloutDanger, CalloutInfo } from "../components/Callout";
import { GenesisData, TimeWindow } from "../lib/types";
import { Page, Role } from "../components/Page";
import { Section } from "../components/Section";
import { SetupSection } from "./types";
import { SetupTiming } from "./Timing";
import { ThumbsDown, ThumbsUp } from "../components/Icons";
import { debounce } from "../lib/debounce";
import { useFoucBarrier } from "../lib/use-fouc-barrier";

export const Election: React.FC = () => {
  return (
    <Page title="Setup a new election" role={Role.Authority}>
      <ElectionData />
    </Page>
  );
};

enum SetupTab {
  Auto = "__auto__",
  BasicInfo = "basic",
  Timing = "timing",
  Candidates = "candidates",
  Trustees = "trustees",
  Registrar = "registrar",
  Genesis = "genesis",
}

const tabs = [
  {
    key: SetupTab.BasicInfo,
    name: "Info",
  },
  {
    key: SetupTab.Timing,
    name: "Timing",
  },
  {
    key: SetupTab.Candidates,
    name: "Candidates",
  },
  {
    key: SetupTab.Registrar,
    name: "Registrar",
  },
  {
    key: SetupTab.Trustees,
    name: "Trustees",
  },
  {
    key: SetupTab.Genesis,
    name: "Genesis",
  },
];

// debounced save, need a falling edge save.
const save = debounce((data: GenesisData) => {
  fetch("/authority/api/config.json", {
    method: "PUT",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(data),
  }).catch((err) => {
    console.log("SAVE FAILED!", err);
  });
}, 500);

const ElectionData: React.FC = () => {
  // load the data upfront.
  const [setup, setSetup] = useState<GenesisData | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [tab, setTab] = useState<SetupTab>(SetupTab.Auto);
  const foucOK = useFoucBarrier();
  useEffect(() => {
    fetch("/authority/api/config.json")
      .then((r) => r.json())
      .then(
        (data) => setSetup(data as GenesisData),
        (err) => setError(err)
      );
  }, []);
  // we will need to update the setup from outside and we will mutate it.
  // it's too deep and complicated to clone everywhere.
  if (error) {
    return <CalloutDanger title="Error loading election data..." />;
  }

  if (!setup) {
    // if less than 100ms then prefer a blank render to a flash of loading
    return foucOK && <CalloutInfo title="Loading... " />;
  }

  const { autotab, progress } = introspectSetupData(setup);
  if (tab === SetupTab.Auto) {
    setTab(autotab);
  }

  const update = (s: GenesisData) => {
    setSetup(s);
    save(s);
  };

  const currentTab = tab;

  return (
    <div className="container">
      <div className="tabs is-medium mt-2 is-boxed">
        <ul>
          {tabs.map(({ key, name }, i) => {
            const icon = progress[key] ? (
              <ThumbsUp className="has-text-primary-dark" />
            ) : (
              <ThumbsDown className="has-text-danger-dark" />
            );
            return (
              <li key={key} className={cx({ "is-active": key === currentTab })}>
                <a onClick={() => setTab(key)}>
                  {i + 1} {name} {icon}
                </a>
              </li>
            );
          })}
        </ul>
      </div>
      {currentTab === SetupTab.BasicInfo && (
        <SetupBasicInfo setup={setup} update={update} />
      )}
      {currentTab === SetupTab.Timing && (
        <SetupTiming setup={setup} update={update} />
      )}
      {currentTab === SetupTab.Candidates && (
        <SetupCandidates setup={setup} update={update} />
      )}
      {currentTab === SetupTab.Registrar && (
        <SetupRegistrar setup={setup} update={update} />
      )}
      {currentTab === SetupTab.Trustees && (
        <SetupTrustees setup={setup} update={update} />
      )}
      {currentTab === SetupTab.Genesis && (
        <SetupGenesis setup={setup} update={update} />
      )}
    </div>
  );
};

const SetupBasicInfo: SetupSection = ({ setup, update }) => {
  return (
    <Section
      title="Step 1: Election Details"
      subtitle="Enter the basic election information."
    >
      <div className="field is-horizontal">
        <div className="field-label is-large">
          <label className="label">Election Name</label>
        </div>
        <div className="field-body">
          <div className="field is-expanded">
            <div className="control">
              <input
                className="input is-large"
                type="text"
                value={setup.name}
                onChange={(evt) => {
                  setup.name = evt.target.value;
                  update({ ...setup });
                }}
              />
            </div>
            <p className="help">This is the display name the voters will see</p>
          </div>
        </div>
      </div>
    </Section>
  );
};

const SetupCandidates: SetupSection = ({ setup, update }) => {
  return (
    <Section
      title="Step 3: List Candidates"
      subtitle="Enumerate the Candidates in this election"
    ></Section>
  );
};

const SetupRegistrar: SetupSection = ({ setup, update }) => {
  return (
    <Section
      title="Step 4: Configure Registrar"
      subtitle="Registrar information and Voter Authentication."
    ></Section>
  );
};

const SetupTrustees: SetupSection = ({ setup, update }) => {
  return (
    <Section
      title="Step 5: Configure Trustees"
      subtitle="Setup the initial data for the Trustees"
    ></Section>
  );
};

const SetupGenesis: SetupSection = ({ setup, update }) => {
  return (
    <Section
      title="Step 6: Create the Genesis block for this election"
      subtitle="Start broadcasting election data"
    ></Section>
  );
};
type Progress = {
  autotab: SetupTab;
  progress: Partial<Record<SetupTab, true>>;
};
/*
"basic",
  Timing = "timing",
  Candidates = "candidates",
  Trustees = "trustees",
  Registrar = "registrar",
  Genesis = "genesis",
}
*/

function bail(): never {
  throw void 0;
}

function introspectSetupData(setup: GenesisData): Progress {
  // look through the data and for each section, if the data is _complete_,
  // mark it in progress and advance autotab.
  const output: Progress = {
    autotab: SetupTab.BasicInfo,
    progress: {},
  };

  try {
    basic();
    output.progress[SetupTab.BasicInfo] = true;
    output.autotab = SetupTab.Timing;
    timings();
    output.progress[SetupTab.Timing] = true;
    output.autotab = SetupTab.Candidates;
    candidates();
    output.progress[SetupTab.Candidates] = true;
    output.autotab = SetupTab.Trustees;
    trustees();
    output.progress[SetupTab.Trustees] = true;
    output.autotab = SetupTab.Registrar;
    registrar();
    output.progress[SetupTab.Registrar] = true;
    output.autotab = SetupTab.Genesis;
  } finally {
    return output;
  }

  function basic() {
    // Basic Info
    if (setup.encryptionSharedParams && setup.name) {
      // OK
      return;
    }
    bail();
  }

  function timings() {
    // Timings
    if (!setup?.timing) {
      bail();
    }
    if (!Info.isValidIANAZone(setup.timing.timeZone)) {
      bail();
    }

    let current: DateTime | null = null;

    checkWindow(setup.timing.parameterConfirmation);
    checkWindow(setup.timing.voterRegistration);
    checkWindow(setup.timing.voteCasting);
    checkWindow(setup.timing.tallyDecryption);

    // we got here we are ok!
    return;

    function checkWindow(w?: TimeWindow) {
      if (!w || !w.opens || !w.closes) {
        bail();
      }
      const odt = DateTime.fromISO(w.opens);
      if (current && odt < current) {
        bail();
      }
      const cdt = DateTime.fromISO(w.closes);
      if (cdt < odt) {
        bail();
      }
      current = cdt;
    }
  }

  function candidates() {
    // must have at least 2 candidates.
    if (!setup.candidates) {
      bail();
    }
    if (setup.candidates.length < 2) {
      bail();
    }
    // each should have a name...
    if (
      setup.candidates.some((c) => {
        c.name.trim() === "";
      })
    ) {
      bail();
    }
  }
  function trustees() {
    // must have at least 3 trustees.
    // threshold set.
    if (!setup.trustees) {
      bail();
    }
    if (setup.trustees.length < 3) {
      bail();
    }
    if (
      setup.trustees.some((t) => {
        // some criteria for the trustee.
        // this will include the initial data back and forth...
        // name must not be emtpy.
        if (t.name.trim() === "") {
          return true;
        }
        // there must be 2 keys a pok and a signature.
        // they must all be correct.
        // this will require a network round trip to validate
        // so we have to wait until the backend has verified it.
        // (the save function will produce the errors...)
        // so we don't do it here... except if they are empty, then we won't accept it.
      })
    ) {
      bail();
    }
  }
  function registrar() {
    // validate data.
    // the name must be given, url, verification key and signature.
    if (!setup.registrar) {
      bail();
    }
    if (setup.registrar.name.trim() === "") {
      bail();
    }
    const u = new URL(setup.registrar.registrationURL || "");
    if (!/^https?$/.test(u.protocol)) {
      bail();
    }
  }
}
