import React, { Component } from 'react';
import WhitePawn from 'pieces/piece01.png';
import WhiteKnight from 'pieces/piece02.png';
import WhiteBishop from'pieces/piece03.png';
import WhiteRook from 'pieces/piece04.png';
import WhiteQueen from 'pieces/piece05.png';
import WhiteKing from 'pieces/piece06.png';
import BlackPawn from 'pieces/piece11.png';
import BlackKnight from 'pieces/piece12.png';
import BlackBishop from 'pieces/piece13.png';
import BlackRook from 'pieces/piece14.png';
import BlackQueen from 'pieces/piece15.png';
import BlackKing from 'pieces/piece16.png';

export default class Board extends Component {
  render() {
    let squares = [];
    const allowedLetters = ['r', 'n', 'b', 'k', 'q', 'p'];
    const images = {
      'R': <img alt="R" src={WhiteRook} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'N': <img alt="R" src={WhiteKnight} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'B': <img alt="R" src={WhiteBishop} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'Q': <img alt="R" src={WhiteQueen} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'K': <img alt="R" src={WhiteKing} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'P': <img alt="R" src={WhitePawn} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'r': <img alt="R" src={BlackRook} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'n': <img alt="R" src={BlackKnight} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'b': <img alt="R" src={BlackBishop} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'q': <img alt="R" src={BlackQueen} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'k': <img alt="R" src={BlackKing} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>,
      'p': <img alt="R" src={BlackPawn} style={{ "width": "60px", "height": "60px", "marginLeft": "20px", "marginTop": "20px" }}/>
    };
    for (let i = 0; i < this.props.position.fen.length && this.props.position.fen[i] !== ' '; i++) {
      if (allowedLetters.includes(this.props.position.fen[i].toLowerCase())) {
        squares.push(this.props.position.fen.charAt(i));
      } else {
        if (this.props.position.fen.charAt(i) !== '/') {
          // its a number
          for(let j = 0; j < parseInt(this.props.position.fen[i], 10); j++) {
            squares.push('');
          }
        }
      }
    }
    let squareDivs = [];
    const colors = ['white', 'gray'];
    let color = 0;
    let index = 0;
    for (let i=0; i <8; i++) {
      for (let j=0; j <8; j++) {
        let border = '1px black solid';
        if (this.props.position && this.props.position.lastMove) {
          if ((63 - index) === this.props.position.lastMove.source || (63 - index) === this.props.position.lastMove.destination) {
            border = '2px yellow solid';
          }
        }
        squareDivs.push(
          <div id={index} key={i.toString() + j.toString()} style={{ 
            "border": border,
            "display": "inline-block",
            "width": "100px",
            "height": "100px",
            "position": "absolute",
            "top": i * 100 + "px",
            "left": j * 100 + "px",
            "backgroundColor": colors[color]
          }}>
            {images[squares[index]]}
          </div>);
        color = (color + 1) % 2;
        index++;
      }
      color = (color + 1) % 2;
    }
    return (
      <div id="board" style={{ 'position':'relative', 'width': '800px', 'height': '800px'}}>
        { squareDivs }
      </div>
    );
  }
}