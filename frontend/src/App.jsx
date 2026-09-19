import { useEffect, useState } from "react";
import "./App.css";

function App() {
  const [showForm, setShowForm] = useState(false);
  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState(["", ""]);
  const [shareUrl, setShareUrl] = useState("");

  const [poll, setPoll] = useState(null);
  const [selectedOption, setSelectedOption] = useState(null);
  const [voted, setVoted] = useState(false);

  // Check whether this is a poll link
  useEffect(() => {
    const path = window.location.pathname;

    if (path.startsWith("/poll/")) {
      const pollId = path.split("/")[2];

      fetchPoll(pollId);
    }
  }, []);

  const fetchPoll = async (pollId) => {
    try {
      const response = await fetch(
        `http://localhost:8080/api/polls/${pollId}`
      );

      const data = await response.json();

      if (response.ok) {
        setPoll(data.poll);
      } else {
        alert("Poll not found.");
      }
    } catch (error) {
      console.error(error);
      alert("Cannot connect to backend.");
    }
  };

  const addOption = () => {
    setOptions([...options, ""]);
  };

  const updateOption = (index, value) => {
    const updatedOptions = [...options];
    updatedOptions[index] = value;
    setOptions(updatedOptions);
  };

  const createPoll = async () => {
    if (!question.trim()) {
      alert("Please enter a poll question.");
      return;
    }

    if (options.some((option) => !option.trim())) {
      alert("Please fill all the options.");
      return;
    }

    try {
      const response = await fetch("http://localhost:8080/api/polls", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          question: question,
          options: options,
        }),
      });

      const data = await response.json();

      if (response.ok) {
        setShareUrl(data.shareUrl);
        alert("Poll created successfully!");
      } else {
        alert("Failed to create poll.");
      }
    } catch (error) {
      console.error(error);
      alert("Cannot connect to backend.");
    }
  };

  const vote = async () => {
    if (selectedOption === null) {
      alert("Please select an option.");
      return;
    }

    const pollId = window.location.pathname.split("/")[2];

    try {
      const response = await fetch(
        `http://localhost:8080/api/polls/${pollId}/vote`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            option: selectedOption,
          }),
        }
      );

      const data = await response.json();

      if (response.ok) {
        setPoll(data.poll);
        setVoted(true);
        alert("Vote recorded successfully!");
      } else {
        alert("Vote failed.");
      }
    } catch (error) {
      console.error(error);
      alert("Cannot connect to backend.");
    }
  };

  // Poll voting page
  if (poll) {
    return (
      <div className="app">
        <nav className="navbar">
          <div className="logo">LivePoll</div>
        </nav>

        <section className="poll-form">
          <h1>{poll.question}</h1>

          <p>Select your answer:</p>

          {poll.options.map((option, index) => (
            <label
              key={index}
              style={{
                display: "block",
                margin: "15px 0",
                padding: "12px",
                cursor: "pointer",
              }}
            >
              <input
                type="radio"
                name="poll"
                checked={selectedOption === index}
                onChange={() => setSelectedOption(index)}
              />

              {" "}{option}
            </label>
          ))}

          {!voted ? (
            <button className="submit-poll-btn" onClick={vote}>
              🗳️ Submit Vote
            </button>
          ) : (
            <div className="share-box">
              <h3>✅ Vote Submitted!</h3>
              <p>Thank you for voting.</p>

              <h3>📊 Current Results</h3>

              {poll.options.map((option, index) => (
                <p key={index}>
                  {option}: {poll.votes[index]} vote(s)
                </p>
              ))}
            </div>
          )}
        </section>
      </div>
    );
  }

  // Home / Create Poll page
  return (
    <div className="app">
      <nav className="navbar">
        <div className="logo">LivePoll</div>

        <div className="nav-links">
          <a href="#">Home</a>

          <a
            href="#"
            onClick={(e) => {
              e.preventDefault();
              setShowForm(true);
              setShareUrl("");
            }}
          >
            Create Poll
          </a>

          <a href="#">Login</a>
        </div>
      </nav>

      {!showForm ? (
        <>
          <section className="hero">
            <h1>Make Every Voice Count</h1>

            <p>
              Create live polls, share them with your audience,
              and watch the results update instantly.
            </p>

            <button
              className="create-btn"
              onClick={() => setShowForm(true)}
            >
              Create Poll
            </button>
          </section>

          <section className="features">
            <div className="feature-card">
              <h3>📝 Create Poll</h3>
              <p>Create questions and add multiple answer options.</p>
            </div>

            <div className="feature-card">
              <h3>🔗 Share Easily</h3>
              <p>Share your poll with your audience using a simple link.</p>
            </div>

            <div className="feature-card">
              <h3>📊 Live Results</h3>
              <p>See voting results update in real time without refreshing.</p>
            </div>
          </section>
        </>
      ) : (
        <section className="poll-form">
          <h1>Create Your Poll</h1>

          <label>Poll Question</label>

          <input
            type="text"
            placeholder="Enter your question"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
          />

          <label>Answer Options</label>

          {options.map((option, index) => (
            <input
              key={index}
              type="text"
              placeholder={`Option ${index + 1}`}
              value={option}
              onChange={(e) => updateOption(index, e.target.value)}
            />
          ))}

          <button className="add-option-btn" onClick={addOption}>
            + Add Option
          </button>

          <button className="submit-poll-btn" onClick={createPoll}>
            Create Poll
          </button>

          {shareUrl && (
            <div className="share-box">
              <h3>🎉 Poll Created!</h3>

              <p>Share this link with your audience:</p>

              <input
                type="text"
                value={shareUrl}
                readOnly
                onClick={(e) => e.target.select()}
              />

              <button
                className="copy-btn"
                onClick={() => {
                  navigator.clipboard.writeText(shareUrl);
                  alert("Share link copied!");
                }}
              >
                📋 Copy Link
              </button>
            </div>
          )}

          <button
            className="back-btn"
            onClick={() => {
              setShowForm(false);
              setShareUrl("");
            }}
          >
            ← Back
          </button>
        </section>
      )}
    </div>
  );
}

export default App;